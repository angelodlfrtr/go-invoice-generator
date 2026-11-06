package facturx

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/angelodlfrtr/go-invoice-generator/invoice"
	"github.com/shopspring/decimal"
)

// Sentinel errors returned (wrapped) by Attach and BuildXML when the document
// or options cannot produce a valid Factur-X invoice. Use errors.Is to test.
var (
	// ErrFixedAmountTax is returned for profiles above MINIMUM when an item has
	// a fixed-amount Tax. EN 16931 only knows rate-based VAT; model such
	// charges as separate items instead.
	ErrFixedAmountTax = errors.New("fixed-amount taxes cannot be represented as VAT in this profile")

	// ErrMissingSellerID is returned when neither SellerTaxID nor SellerLegalID
	// is set (BR-CO-26).
	ErrMissingSellerID = errors.New("seller needs a VAT ID (SellerTaxID) or a legal registration ID (SellerLegalID)")

	// ErrMissingCountry is returned when a required country code is missing.
	ErrMissingCountry = errors.New("country code is required")

	// ErrInvalidCountryCode is returned for a country code that is not two
	// letters (ISO 3166-1 alpha-2).
	ErrInvalidCountryCode = errors.New("invalid ISO 3166-1 alpha-2 country code")

	// ErrInvalidCurrencyCode is returned for a currency code that is not three
	// letters (ISO 4217).
	ErrInvalidCurrencyCode = errors.New("invalid ISO 4217 currency code")

	// ErrInvalidDate is invoice.ErrInvalidDate: a date that is not DD/MM/YYYY,
	// YYYY-MM-DD or YYYYMMDD.
	ErrInvalidDate = invoice.ErrInvalidDate

	// ErrInvalidTaxCategory is returned for an unknown VAT category code or a
	// rate that does not match the category (BR-S-05, BR-E-05, BR-O-05...).
	ErrInvalidTaxCategory = errors.New("invalid VAT category")

	// ErrMissingExemptionReason is returned when a VAT category other than S or
	// Z has neither an exemption reason nor a reason code (BR-E-10, BR-AE-10...).
	ErrMissingExemptionReason = errors.New("VAT exemption reason or reason code is required")

	// ErrMissingPaymentTerms is returned for profiles above MINIMUM when an
	// amount is due but neither a due date nor payment terms are given (BR-CO-25).
	ErrMissingPaymentTerms = errors.New("a payment due date or payment terms description is required when an amount is due")

	// ErrUnsupportedDocumentType is returned for quotations and delivery notes,
	// which have no e-invoice representation, unless Options.TypeCode is set.
	ErrUnsupportedDocumentType = errors.New("document type has no Factur-X representation")

	// ErrInvalidPeriod is returned when only one of InvoicingPeriodStart and
	// InvoicingPeriodEnd is set.
	ErrInvalidPeriod = errors.New("invoicing period needs both a start and an end date")

	// ErrUnsupportedPrecision is returned when Options.CurrencyPrecision is
	// above 2: EN 16931 amounts carry at most two decimals (BR-DEC).
	ErrUnsupportedPrecision = errors.New("currency precision above 2 decimals is not supported")

	// ErrInvalidVATPointDateCode is returned for a VATPointDateCode other
	// than "5", "29" or "72".
	ErrInvalidVATPointDateCode = errors.New("invalid VAT point date code (expected 5, 29 or 72)")

	// ErrInvalidTaxCurrency is returned when TaxCurrencyCode is malformed,
	// equals the invoice currency, or has no positive TaxCurrencyRate.
	ErrInvalidTaxCurrency = errors.New("invalid tax currency")
)

var (
	countryCodeRe  = regexp.MustCompile(`^[A-Z]{2}$`)
	currencyCodeRe = regexp.MustCompile(`^[A-Z]{3}$`)
)

// vatPointDateCodes are the UNTDID 2475 values allowed for BT-8.
var vatPointDateCodes = map[string]bool{"5": true, "29": true, "72": true}

// VAT category codes (UNTDID 5305 subset used by EN 16931).
var vatCategories = map[string]struct {
	needsReason bool
	zeroRate    bool // rate must be 0
	noRate      bool // rate must be absent
}{
	"S":  {},
	"Z":  {zeroRate: true},
	"E":  {needsReason: true, zeroRate: true},
	"AE": {needsReason: true, zeroRate: true},
	"K":  {needsReason: true, zeroRate: true},
	"G":  {needsReason: true, zeroRate: true},
	"O":  {needsReason: true, noRate: true},
	"L":  {},
	"M":  {},
}

// lineTax is the VAT information resolved for one item.
type lineTax struct {
	category            string
	percent             decimal.Decimal
	hasRate             bool
	exemptionReason     string
	exemptionReasonCode string
	fixedAmount         bool // generator fixed-amount tax, only legal in MINIMUM
}

// resolveLineTax maps an item's Tax to EN 16931 VAT semantics.
func resolveLineTax(item *invoice.Item, opts Options) (lineTax, error) {
	return resolveTax("item "+item.Name, item.Tax, opts)
}

// resolveTax maps a Tax (nil for untaxed) to EN 16931 VAT semantics and
// checks the category against its rate. what names the owner in errors.
func resolveTax(what string, tax *invoice.Tax, opts Options) (lineTax, error) {
	lt := lineTax{hasRate: true}

	switch {
	case tax == nil:
		lt.category = opts.noTaxCategoryCode()
		lt.percent = decimal.Zero
		lt.exemptionReason = opts.NoTaxExemptionReason
		lt.exemptionReasonCode = opts.NoTaxExemptionReasonCode

	case tax.Amount != "":
		lt.fixedAmount = true
		return lt, nil

	default:
		p, err := decimal.NewFromString(tax.Percent)
		if err != nil {
			return lt, fmt.Errorf("%s: tax percent %q: %w", what, tax.Percent, err)
		}
		lt.percent = p
		lt.category = tax.Category
		if lt.category == "" {
			lt.category = opts.taxCategoryCode()
		}
		lt.exemptionReason = tax.ExemptionReason
		lt.exemptionReasonCode = tax.ExemptionReasonCode
	}

	lt.category = strings.ToUpper(strings.TrimSpace(lt.category))
	spec, ok := vatCategories[lt.category]
	if !ok {
		return lt, fmt.Errorf("%s: category %q: %w", what, lt.category, ErrInvalidTaxCategory)
	}
	switch {
	case spec.noRate:
		if !lt.percent.IsZero() {
			return lt, fmt.Errorf("%s: category %s must not carry a rate, got %s: %w", what, lt.category, lt.percent, ErrInvalidTaxCategory)
		}
		lt.hasRate = false
	case spec.zeroRate && !lt.percent.IsZero():
		return lt, fmt.Errorf("%s: category %s requires a 0%% rate, got %s: %w", what, lt.category, lt.percent, ErrInvalidTaxCategory)
	case lt.category == "S" && lt.percent.Sign() <= 0:
		return lt, fmt.Errorf("%s: category S requires a rate above 0%% (use Z or E for untaxed items): %w", what, ErrInvalidTaxCategory)
	}
	if spec.needsReason && lt.exemptionReason == "" && lt.exemptionReasonCode == "" {
		return lt, fmt.Errorf("%s: category %s: %w", what, lt.category, ErrMissingExemptionReason)
	}

	return lt, nil
}

// normalizeCountry upper-cases and validates an ISO 3166-1 alpha-2 code.
func normalizeCountry(field, code string, required bool) (string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		if required {
			return "", fmt.Errorf("%s: %w", field, ErrMissingCountry)
		}
		return "", nil
	}
	if !countryCodeRe.MatchString(code) {
		return "", fmt.Errorf("%s %q: %w", field, code, ErrInvalidCountryCode)
	}
	return code, nil
}

// formatDate converts a document date (see invoice.ParseDate) to "YYYYMMDD"
// (CII format 102). It returns ErrInvalidDate for anything else.
func formatDate(field, date string) (string, error) {
	t, err := invoice.ParseDate(date)
	if err != nil {
		return "", fmt.Errorf("%s: %w", field, err)
	}
	return t.Format("20060102"), nil
}

// optionalDate is formatDate for fields that may be empty.
func optionalDate(field, date string) (string, error) {
	if strings.TrimSpace(date) == "" {
		return "", nil
	}
	return formatDate(field, date)
}
