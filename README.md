# go-invoice-generator

Build invoices, quotations, delivery notes and credit notes in Go: a data model
with EN 16931 arithmetic, a PDF renderer, and a Factur-X / ZUGFeRD e-invoice
exporter validated against the European and French rule sets.

## Features

- Four document types: Invoice, Quotation, Delivery Note, Credit Note
- Per-item tax and discount (percentage or fixed amount), document-level discount and charges, prepayments, default tax
- Named taxes with a per-name breakdown in the totals block
- EN 16931 rounding model: totals are sums of rounded parts, identical on the PDF and in the e-invoice
- Party identifiers (VAT number, SIREN, electronic address, contact person), delivery, payment instructions and preceding invoice, printed on the PDF and exported
- Custom header and footer with optional pagination, customisable labels, colours and currency formatting
- Unicode support via a configurable translation function; Roboto embedded by default
- Output to file or `[]byte`
- [Factur-X](#factur-x) subpackage producing PDF/A-3B e-invoices for all five profiles, with a French e-invoicing mode, verified with veraPDF and mustangproject (EN 16931 and BR-FR schematrons)
- Reception: extract and parse a received Factur-X / ZUGFeRD PDF back into a document and verify its totals

---

## v0.8.0 — Breaking changes

The data model moved out of the renderer into the `invoice` package:

```diff
-import generator "github.com/angelodlfrtr/go-invoice-generator/generator"
+import (
+    "github.com/angelodlfrtr/go-invoice-generator/generator"
+    "github.com/angelodlfrtr/go-invoice-generator/invoice"
+)
```

- `generator.New(type, opts)` still exists and returns a `generator.Document` that embeds an `invoice.Document`, so the fluent setters and `Build()` keep working. `generator.Contact`, `Item`, `Tax`, `Discount` and the document type constants are aliases of the `invoice` types for one release; new code should use `invoice.New` and `generator.NewGenerator`.
- `Options.CurrencyPrecision` is gone: precision lives in `invoice.Document.Currency` (EUR, 2 decimals by default).
- `Contact.AddtionnalInfo` is renamed `AdditionalInfo`.
- `Document.Notes` is a `[]invoice.Note` (content plus optional subject code). `SetNotes(string)` still sets a single note.
- Totals follow the EN 16931 rounding model (see [Rounding model](#rounding-model)); some multi-rate or discounted invoices change by a cent compared with v0.7.0.
- `facturx.Attach` and `BuildXML` take an `*invoice.Document`. Party, delivery, payment and note options moved from `facturx.Options` to the document; `facturx.Options` only keeps format settings.

---

## Installation

```sh
go get github.com/angelodlfrtr/go-invoice-generator
```

## Quick start

```go
package main

import (
	"log"

	"github.com/angelodlfrtr/go-invoice-generator/generator"
	"github.com/angelodlfrtr/go-invoice-generator/invoice"
)

func main() {
	doc, err := invoice.New(invoice.Invoice)
	if err != nil {
		log.Fatal(err)
	}

	doc.SetRef("INV-2024-001")
	doc.SetDate("01/01/2024")
	doc.SetDueDate("31/01/2024")

	doc.SetCompany(&invoice.Contact{
		Name:  "Acme Corp",
		VATID: "FR12345678901",
		Address: &invoice.Address{
			Address:     "1 Rue de la Paix",
			PostalCode:  "75001",
			City:        "Paris",
			Country:     "France",
			CountryCode: "FR",
		},
	})

	doc.SetCustomer(&invoice.Contact{
		Name: "John Doe",
		Address: &invoice.Address{
			Address:    "42 Main Street",
			PostalCode: "10001",
			City:       "New York",
			Country:    "US",
		},
	})

	doc.AppendItem(&invoice.Item{
		Name:     "Consulting",
		UnitCost: "150.00",
		Quantity: "8",
		Tax:      &invoice.Tax{Percent: "20"},
	})

	pdf, err := generator.NewGenerator(&generator.Options{
		TextTypeInvoice: "INVOICE",
		CurrencySymbol:  "€ ",
	}).Build(doc)
	if err != nil {
		log.Fatal(err)
	}

	if err := pdf.OutputFileAndClose("invoice.pdf"); err != nil {
		log.Fatal(err)
	}
}
```

The one-value form from earlier versions is still available:

```go
doc, _ := generator.New(generator.Invoice, &generator.Options{})
doc.SetRef("INV-2024-001") // all invoice.Document setters are promoted
pdf, err := doc.Build()
```

## Example output

![DeliveryNoteExample](support/example.png)

---

## Packages

| Package     | Role                                                                                                                                             |
| ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| `invoice`   | The document model (parties, items, taxes, discounts, delivery, payment, notes), validation, dates and the totals arithmetic. No PDF dependency. |
| `generator` | Renders an `invoice.Document` to PDF with fpdf: labels, colours, fonts, header and footer.                                                       |
| `facturx`   | Embeds a Factur-X / ZUGFeRD CII XML derived from the document into the PDF and makes it PDF/A-3B.                                                |

---

## Document types

| Constant               | Value             | Factur-X type code |
| ---------------------- | ----------------- | ------------------ |
| `invoice.Invoice`      | `"INVOICE"`       | 380                |
| `invoice.CreditNote`   | `"CREDIT_NOTE"`   | 381                |
| `invoice.Quotation`    | `"QUOTATION"`     | —                  |
| `invoice.DeliveryNote` | `"DELIVERY_NOTE"` | —                  |

```go
doc, err := invoice.New(invoice.Quotation)
```

---

## Document fields

Everything below is data; the generator prints it and the Factur-X exporter maps it to the corresponding EN 16931 business terms (BT).

```go
doc.SetRef("INV-2024-001")              // document number (BT-1)
doc.SetVersion("2")
doc.SetDate("01/01/2024")               // issue date (BT-2)
doc.SetDueDate("31/01/2024")            // payment due date (BT-9)
doc.SetPaymentTerm("30 days, 2% early") // free-text payment terms (BT-20)
doc.SetClientRef("COST-CENTRE-42")      // buyer reference (BT-10)
doc.SetPurchaseOrderRef("PO-99")        // purchase order (BT-13)
doc.SetDescription("Services for December")
doc.SetCurrency("EUR", 2)               // ISO 4217 code and precision

doc.SetPayment(&invoice.Payment{        // payment instructions (BG-16)
	IBAN:      "FR7630006000011234567890189",
	BIC:       "BNPAFRPP",
	Reference: "INV-2024-001",          // remittance information (BT-83)
})

doc.SetDelivery(&invoice.Delivery{      // delivery (BG-13, BG-14)
	Date:        "2023-12-31",
	PeriodStart: "01/12/2023",
	PeriodEnd:   "31/12/2023",
	ShipTo: &invoice.Contact{Name: "Warehouse", Address: &invoice.Address{/* ... */}},
})

doc.SetPrecedingInvoice("INV-2023-120", "2023-12-01") // for credit notes (BG-3)

doc.AddCharge(&invoice.Charge{                // document-level charge (BG-21)
	Name: "Shipping", Amount: "15.00", ReasonCode: "FC", Tax: &invoice.Tax{Percent: "20"},
})
doc.SetPrepaid("100.00")                      // already paid (BT-113); the amount due nets it

doc.SetNotes("Thank you for your business.")           // one note, basic HTML allowed
doc.AddNote("Pas d'escompte pour paiement anticipé", "AAB") // note with UNTDID 4451 subject code
```

Dates accept `DD/MM/YYYY`, `YYYY-MM-DD` or `YYYYMMDD` (`invoice.ParseDate`). When `DueDate` is empty and `PaymentTerm` is a date, it is used as the due date.

---

## Contacts

The company and the customer are `invoice.Contact` values. A logo can be embedded as a `[]byte` (PNG or JPEG). Identifiers are printed under the address and exported to the e-invoice.

```go
doc.SetCompany(&invoice.Contact{
	Name:  "Acme Corp",
	Logo:  logoBytes,
	Address: &invoice.Address{
		Address:     "1 Market Street",
		Address2:    "Suite 200",
		PostalCode:  "94105",
		City:        "San Francisco",
		Country:     "USA",     // printed as given
		CountryCode: "US",      // ISO 3166-1 alpha-2, used by exporters
	},
	VATID:         "FR12345678901",   // VAT registration (BT-31)
	LegalID:       "123456789",       // legal registration, e.g. SIREN (BT-30)
	LegalIDScheme: "0002",            // ISO 6523 ICD of the register
	ElectronicAddress:       "billing@acme.example", // e-invoicing endpoint (BT-34)
	ElectronicAddressScheme: "EM",
	ContactName:  "Jane Smith",       // contact person (BG-6)
	ContactPhone: "+33 1 23 45 67 89",
	ContactEmail: "jane@acme.example",
	AdditionalInfo: []string{"RCS Paris 123 456 789", "Capital 10 000 €"}, // free lines, basic HTML
})
```

`CountryCode` may be omitted when `Country` already is a two-letter code.

---

## Items

```go
doc.AppendItem(&invoice.Item{
	Name:        "Web development",
	Description: "Frontend and backend implementation",
	UnitCost:    "1200.00",
	Quantity:    "3",
	Unit:        "DAY",              // UN/ECE Rec. 20 unit code, default "C62" (piece)
	Tax:         &invoice.Tax{Percent: "20"},
	Discount:    &invoice.Discount{Percent: "10"},
})
```

`UnitCost` and `Quantity` are strings to avoid floating-point precision issues;
the library uses [shopspring/decimal](https://github.com/shopspring/decimal) internally.

### Tax

A tax is either a **percentage** or a **fixed amount**, not both. An optional `Name`
groups taxes in the totals breakdown.

```go
Tax: &invoice.Tax{Percent: "20", Name: "VAT 20%"}
Tax: &invoice.Tax{Percent: "5.5", Name: "VAT 5.5%"}
Tax: &invoice.Tax{Amount: "89", Name: "Eco tax"}   // fixed amount regardless of quantity
Tax: &invoice.Tax{Percent: "10"}                   // unnamed, grouped under "Other"
```

When two or more distinct tax names are present the totals block shows the overall
tax amount followed by a smaller per-name breakdown. Taxes without a name are grouped
under the label set by `Options.TextTotalTaxOther` (default: `"Other"`).

For e-invoicing a tax also carries its EN 16931 VAT category and, when the category
requires it, an exemption reason. These fields do not change the rendered PDF.

```go
// Exempt line: category E needs a reason or a VATEX reason code
Tax: &invoice.Tax{
	Percent:             "0",
	Category:            "E",
	ExemptionReason:     "TVA non applicable, art. 293 B du CGI",
	ExemptionReasonCode: "VATEX-FR-FRANCHISE",
}

// Intra-community supply (reverse charge by the buyer)
Tax: &invoice.Tax{Percent: "0", Category: "K", ExemptionReason: "Intra-community supply"}
```

### Discount

A discount is either a **percentage** or a **fixed amount**, not both.

```go
Discount: &invoice.Discount{Percent: "30"}  // 30% off the item subtotal
Discount: &invoice.Discount{Amount: "50"}   // fixed amount off the item subtotal
```

---

## Default tax

`SetDefaultTax` applies a tax to every item that does not have its own `Tax`.

```go
doc.SetDefaultTax(&invoice.Tax{Percent: "20"})
```

---

## Document-level discount

A document discount is applied to the subtotal after all item discounts. It
reduces both the pre-tax total and, proportionally per tax group, the tax due.

```go
doc.SetDiscount(&invoice.Discount{Amount: "500"})
doc.SetDiscount(&invoice.Discount{Percent: "5"})
```

---

## Totals

All totals are available after `Validate()`, which `Build()` runs for you.

```go
if err := doc.Validate(); err != nil {
	log.Fatal(err)
}

fmt.Println(doc.TotalWithoutTaxAndWithoutDocumentDiscount()) // sum of item subtotals after item discounts
fmt.Println(doc.DocumentDiscount())                         // document discount amount
fmt.Println(doc.TotalWithoutTax())                          // above minus document discount
fmt.Println(doc.Tax())                                      // total tax
fmt.Println(doc.ChargeTotal())                              // document-level charges
fmt.Println(doc.TotalWithTax())                             // grand total
fmt.Println(doc.PrepaidAmount())                            // already paid
fmt.Println(doc.AmountDue())                                // grand total minus prepaid
fmt.Println(doc.TaxLines())                                 // per-name tax lines, nil when no tax is named
```

Item-level helpers:

```go
item := &invoice.Item{UnitCost: "100", Quantity: "2", Discount: &invoice.Discount{Percent: "10"}}
if err := item.Prepare(); err != nil {
	log.Fatal(err)
}

fmt.Println(item.TotalWithoutTaxAndWithoutDiscount()) // 200.00
fmt.Println(item.TotalWithoutTaxAndWithDiscount())    // 180.00
fmt.Println(item.TaxWithTotalDiscounted())            // 0 (no tax set)
fmt.Println(item.TotalWithTaxAndDiscount())           // 180.00
```

### Rounding model

Totals follow the EN 16931 model used by accounting software and e-invoicing
validators. Amounts are rounded to `Currency.Precision` at three points:

1. each item's net amount after its own discount, and each document charge;
2. the document discount, which is then split across tax groups in proportion to
   their line totals (the rounding residual goes to the largest group);
3. each tax group's tax, computed on the group's rounded taxable amount.

A tax group is the set of items and charges sharing a `Tax.Category` and rate; fixed-amount
taxes and untaxed items form their own groups. Every total is a sum of those
rounded parts, never a rounded sum, so the totals block, the per-name tax lines
and any Factur-X export derived from the document agree to the cent.

`doc.Breakdown()` exposes the whole computation: the rounded line amounts, the
groups with their line total, discount share, taxable amount and tax, and the
document totals.

```go
for _, g := range doc.Breakdown().Groups {
	fmt.Println(g.Category, g.Percent, g.Basis(), g.Tax)
}
```

---

## Rendering options

All fields are optional and have sensible defaults.

```go
gen := generator.NewGenerator(&generator.Options{
	// Currency formatting (precision comes from doc.Currency)
	CurrencySymbol:   "€ ",  // default: "€ "
	CurrencyDecimal:  ".",   // default: "."
	CurrencyThousand: " ",   // default: " "

	// Localised labels
	TextTypeInvoice:           "INVOICE",
	TextTypeQuotation:         "QUOTATION",
	TextTypeDeliveryNote:      "DELIVERY NOTE",
	TextTypeCreditNote:        "CREDIT NOTE",
	TextRefTitle:              "Ref.",
	TextVersionTitle:          "Version",
	TextDateTitle:             "Date",
	TextDueDateTitle:          "Due date",
	TextPaymentTermTitle:      "Payment term",
	TextPaymentReferenceTitle: "Payment ref.",
	TextPrecedingInvoiceTitle: "Refers to invoice",
	TextDeliveryDateTitle:     "Delivery date",
	TextDeliveryPeriodTitle:   "Period",
	TextDeliveredToTitle:      "Delivered to",
	TextContactVATID:          "VAT",
	TextContactLegalID:        "Reg. no.",
	TextItemsNameTitle:        "Name",
	TextItemsUnitCostTitle:    "Unit price",
	TextItemsQuantityTitle:    "Qty",
	TextItemsTotalHTTitle:     "Total no tax",
	TextItemsTaxTitle:         "Tax",
	TextItemsDiscountTitle:    "Discount",
	TextItemsTotalTTCTitle:    "Total",
	TextTotalTotal:            "Total",
	TextTotalDiscounted:       "Total discounted",
	TextTotalTax:              "Tax",
	TextTotalTaxOther:         "Other",
	TextTotalWithTax:          "Total with tax",

	// Colours (RGB)
	BaseTextColor: []int{35, 35, 35},
	GreyTextColor: []int{82, 82, 82},
	GreyBgColor:   []int{232, 232, 232},
	DarkBgColor:   []int{212, 212, 212},

	// Font family name. Roboto is embedded and used by default; any font
	// registered on the underlying fpdf instance can be used here.
	Font:     "Roboto",
	BoldFont: "Roboto",
})
```

A `Generator` renders one document: create a new one per PDF. A second `Build` returns `ErrAlreadyBuilt`.

---

## Header and footer

```go
gen.SetHeader(&generator.HeaderFooter{
	Text:       "<center>Acme Corp — Confidential</center>",
	FontSize:   7,
	Pagination: true, // show "Page X/{nb}" in the top-right corner
})

gen.SetFooter(&generator.HeaderFooter{
	Text:       "<center>Acme Corp · 1 Market Street · San Francisco</center>",
	Pagination: true,
})
```

`Text` supports basic HTML tags (`<b>`, `<i>`, `<center>`).

For full control you can provide a custom function directly on the underlying
fpdf instance:

```go
hf := &generator.HeaderFooter{UseCustomFunc: true}
hf.ApplyFunc(gen.Pdf(), func() {
	// use gen.Pdf() (a *fpdf.Fpdf) to draw anything you want
})
gen.SetHeader(hf)
```

---

## Unicode support

The embedded Roboto fonts are UTF-8, so strings are written as-is. To use
another encoding with a non-UTF-8 font (e.g. ISO-8859-2), register the font on
`gen.Pdf()` and install a translator with `gen.SetUnicodeTranslator`; see
`examples/iso_8859_2_cp`.

---

## Output

**To a file:**

```go
pdf, err := gen.Build(doc)
if err != nil {
	log.Fatal(err)
}
if err := pdf.OutputFileAndClose("out.pdf"); err != nil {
	log.Fatal(err)
}
```

**To a byte slice (e.g. for an HTTP response or S3 upload):**

```go
var buf bytes.Buffer
if err := pdf.Output(&buf); err != nil {
	log.Fatal(err)
}
// buf.Bytes() contains the PDF
```

---

## Factur-X

The `facturx` subpackage embeds a [Factur-X](https://fnfe-mpe.org/factur-x/) (also known as ZUGFeRD 2.x) CII XML derived from the document into the PDF produced by the generator, and brings the file to PDF/A-3B. All five conformance levels are supported and validated with mustang-cli (profile XSDs, EN 16931 schematron and, for French documents, the BR-FR schematron) and veraPDF. `Attach` also:

- sets `/AFRelationship /Alternative` on the embedded file, as required by PDF/A-3;
- inserts an sRGB ICC OutputIntent into the PDF catalog (pure Go, no external dependencies);
- merges the required Factur-X `pdfaid`, `pdfaExtension` and `fx:` XMP declarations into the PDF's existing XMP packet without discarding fields written by fpdf.

```go
import (
    "bytes"

    "github.com/angelodlfrtr/go-invoice-generator/facturx"
    "github.com/angelodlfrtr/go-invoice-generator/generator"
    "github.com/angelodlfrtr/go-invoice-generator/invoice"
)

// 1. Build the PDF as usual.
pdf, err := generator.NewGenerator(nil).Build(doc)
if err != nil {
    log.Fatal(err)
}

var buf bytes.Buffer
if err := pdf.Output(&buf); err != nil {
    log.Fatal(err)
}

// 2. Attach the Factur-X XML and bring the document into PDF/A-3B conformance.
result, err := facturx.Attach(buf.Bytes(), doc, facturx.Options{
    Profile: facturx.ProfileEN16931,
})
if err != nil {
    log.Fatal(err)
}

os.WriteFile("invoice_facturx.pdf", result, 0644)
```

`facturx.BuildXML(doc, opts)` returns the bare CII XML without touching the PDF.

### Profiles

| Constant                  | Factur-X profile | Carries                                                               |
| ------------------------- | ---------------- | --------------------------------------------------------------------- |
| `facturx.ProfileMinimum`  | MINIMUM          | Parties (name, country, identifiers), references, totals              |
| `facturx.ProfileBasicWL`  | BASIC-WL         | + full addresses, VAT breakdown, allowances, payment, delivery, notes |
| `facturx.ProfileBasic`    | BASIC            | + line items                                                          |
| `facturx.ProfileEN16931`  | EN 16931         | + item descriptions, seller contact, BIC                              |
| `facturx.ProfileExtended` | EXTENDED         | Same content as EN 16931                                              |

Document data belonging to a higher profile than the one selected is silently omitted from the XML.

Only invoices and credit notes have an e-invoice representation; a quotation or delivery note is rejected with `ErrUnsupportedDocumentType` unless `Options.TypeCode` is set explicitly.

### What comes from the document

| Document field                                            | EN 16931                                        |
| --------------------------------------------------------- | ----------------------------------------------- |
| `Ref`, `Date`, `Type`                                     | BT-1, BT-2, BT-3 (380 invoice, 381 credit note) |
| `ClientRef`, `PurchaseOrderRef`                           | BT-10, BT-13                                    |
| `Currency.Code`                                           | BT-5                                            |
| `Company` / `Customer` name, address, `CountryCode`       | BG-4 / BG-7, BG-5 / BG-8                        |
| `Contact.VATID`, `LegalID` + `LegalIDScheme`              | BT-31 / BT-48, BT-30 / BT-47                    |
| `Contact.ElectronicAddress` + scheme                      | BT-34 / BT-49                                   |
| `Company.ContactName`, `ContactPhone`, `ContactEmail`     | BG-6                                            |
| `Delivery.Date`, `PeriodStart`, `PeriodEnd`, `ShipTo`     | BT-72, BG-14, BG-13 / BG-15                     |
| `DueDate` or a dated `PaymentTerm`; textual `PaymentTerm` | BT-9; BT-20                                     |
| `Payment.IBAN`, `BIC`, `MeansCode`, `Reference`           | BG-16 (BT-84, BT-86, BT-81, BT-83)              |
| `PrecedingInvoice`                                        | BG-3                                            |
| `Notes` (content, subject code)                           | BG-1                                            |
| `Item.Unit`, `Tax.Category`, `ExemptionReason`            | BT-130, BT-151, BT-120 / BT-121                 |

### How the XML is derived

- **VAT breakdown.** Items are grouped by VAT category and rate (BR-CO-18). The category comes from `Tax.Category`, falling back to `TaxCategoryCode` (default `"S"`). Items without a `Tax` use `NoTaxCategoryCode` (default `"E"`, exempt) and must be given a reason through `NoTaxExemptionReason` or `NoTaxExemptionReasonCode`.
- **Document discount.** `doc.Discount` becomes one document-level allowance per VAT group (reason code `95`), split in proportion to each group's line total.
- **Charges.** Each `doc.Charges` entry becomes a document-level charge (BG-21) with its reason code, VAT category and rate, and joins the VAT group of its tax.
- **Totals.** Every amount comes from `doc.Breakdown()`, the same figures printed on the PDF (see [Rounding model](#rounding-model)), so the XML matches the PDF to the cent and the EN 16931 arithmetic rules hold exactly (BR-CO-10, 11, 13, 14, 15). Amounts carry two decimals, so a currency precision above 2 is rejected.
- **Fixed-amount taxes** (`Tax.Amount`) have no VAT representation. They are accepted for MINIMUM, which carries no breakdown, and rejected with `ErrFixedAmountTax` above it; model such charges as separate items instead.

### Validation

`Attach` and `BuildXML` reject documents that cannot produce a valid invoice instead of emitting XML that validators will refuse. Errors wrap one of the sentinels below; test them with `errors.Is`.

| Error                        | Raised when                                                                              |
| ---------------------------- | ---------------------------------------------------------------------------------------- |
| `ErrMissingSellerID`         | The company has neither a `VATID` nor a `LegalID` (BR-CO-26)                             |
| `ErrMissingCountry`          | Seller country missing, or buyer / ship-to country missing above MINIMUM                 |
| `ErrInvalidCountryCode`      | A country code is not two letters (ISO 3166-1 alpha-2)                                   |
| `ErrInvalidCurrencyCode`     | The currency code is not three letters (ISO 4217)                                        |
| `ErrInvalidDate`             | A date does not parse                                                                    |
| `ErrInvalidTaxCategory`      | Unknown category, `S` with a 0% rate, or an exempt category with a non-zero rate         |
| `ErrMissingExemptionReason`  | A category other than `S` or `Z` has no reason and no reason code                        |
| `ErrFixedAmountTax`          | An item has a fixed-amount tax and the profile is above MINIMUM                          |
| `ErrMissingPaymentTerms`     | Above MINIMUM, an amount is due but there is no due date and no payment terms (BR-CO-25) |
| `ErrInvalidPeriod`           | Only one of the invoicing period dates is set                                            |
| `ErrInvalidVATPointDateCode` | `VATPointDateCode` is not `5`, `29` or `72`                                              |
| `ErrInvalidTaxCurrency`      | Tax currency malformed, equal to the invoice currency, or without a positive rate        |
| `ErrUnsupportedPrecision`    | `Currency.Precision` is above 2                                                          |
| `ErrUnsupportedDocumentType` | The document is a quotation or delivery note and no `TypeCode` override is given         |
| `ErrFrenchRules`             | `FrenchRules` is set and a French requirement is broken                                  |

### French e-invoicing

Set `FrenchRules: true` to produce documents for the French e-invoicing reform (AFNOR XP Z12-012, "Flux 2"). On top of EN 16931 the library then:

- requires profile `EN 16931` or `EXTENDED`, a company in France with its SIREN in `LegalID` (scheme `"0002"`), and `BusinessProcessID` set to the cadre de facturation (`B1`, `S1`, `M1`, `B2`, `S2`, `M2`, `B4`, `S4`, `M4`, `S5`, `S6`, `B7`, `S7`); an already-paid cadre (`B2`, `S2`, `M2`) needs `Prepaid` equal to the total and a `DueDate`, and a final invoice after deposit (`B4`, `S4`, `M4`) cannot be a deposit invoice;
- requires the three legal mentions as notes with subject codes `PMD` (pénalités de retard), `PMT` (indemnité forfaitaire de recouvrement) and `AAB` (escompte ou absence d'escompte), each exactly once; they are printed on the PDF like any note;
- adds a `BAR` note carrying `FrenchFlow` (`B2B` by default) and, for `B2B`, requires the customer's SIREN;
- defaults the electronic addresses to the SIREN with the French routing scheme `"0225"` when none is given, and rejects e-mail style addresses;
- checks the document type against the French list, French VAT rates only, no `L`/`M` categories, dates between 2000 and 2099, due date not before the document date, at most 4 decimals on quantities, `EUR` as tax currency for foreign-currency invoices, and a preceding invoice with its date on credit notes and corrective invoices.

```go
doc, _ := invoice.New(invoice.CreditNote)
doc.SetRef("AV-2024-001").SetDate("2024-01-15").SetDueDate("2024-02-15")
doc.SetPrecedingInvoice("INV-2023-120", "2023-12-01")
doc.SetCompany(&invoice.Contact{
	Name: "Acme SAS", VATID: "FR12345678901", LegalID: "123456789", LegalIDScheme: "0002",
	Address: &invoice.Address{Address: "1 Rue de la Paix", PostalCode: "75001", City: "Paris", Country: "France", CountryCode: "FR"},
})
doc.SetCustomer(&invoice.Contact{
	Name: "Client SARL", LegalID: "987654321", LegalIDScheme: "0002",
	Address: &invoice.Address{Address: "5 Rue de la République", PostalCode: "69001", City: "Lyon", CountryCode: "FR"},
})
doc.AddNote("Pénalités de retard : 3 fois le taux d'intérêt légal", "PMD")
doc.AddNote("Indemnité forfaitaire pour frais de recouvrement : 40 €", "PMT")
doc.AddNote("Pas d'escompte pour paiement anticipé", "AAB")
// ... items ...

result, err := facturx.Attach(pdfBytes, doc, facturx.Options{
	Profile:           facturx.ProfileEN16931,
	FrenchRules:       true,
	BusinessProcessID: "S1", // services, standard case
	VATPointDateCode:  "72", // TVA sur les encaissements
})
```

Violations are reported before any XML is produced and wrap `ErrFrenchRules`. The test suite validates the French fixture with mustangproject 2.26+, which applies the official BR-FR schematron when both parties are in France. Transmission through an accredited platform, e-reporting and archiving remain outside the library's scope.

### Reading received invoices

The reception side mirrors the writer: `Extract` pulls the XML out of a Factur-X / ZUGFeRD 2 PDF, `Parse` turns the CII XML into an `invoice.Document` plus the Factur-X metadata and the totals exactly as the sender stated them, and `Read` does both.

```go
p, err := facturx.Read(pdfBytes)
if err != nil {
    log.Fatal(err)
}

fmt.Println(p.Profile, p.TypeCode, p.BusinessProcessID, p.FrenchFlow)
fmt.Println(p.Document.Ref, p.Document.Company.VATID, p.Totals.DuePayable)

// Recompute the totals from the parsed lines and compare them with the stated ones.
if err := p.Verify(); err != nil {
    log.Fatal(err) // wraps facturx.ErrTotalsMismatch
}

// The parsed document renders and exports like any other.
pdf, _ := generator.NewGenerator(nil).Build(p.Document)
```

`Parse` accepts every profile. Below BASIC the XML carries no lines, so `Document.Items` is empty and `Verify` only checks the identities between the stated totals. Document-level allowances collapse into `Document.Discount`; charges, prepaid amount, parties, delivery, payment, notes and the preceding invoice are restored field by field. A French `BAR` note is reported as `FrenchFlow` rather than kept as a note, so the document can be re-exported under `FrenchRules`.

### Options

| Field                                                                   | Profile  | Description                                                                                   |
| ----------------------------------------------------------------------- | -------- | --------------------------------------------------------------------------------------------- |
| `Profile`                                                               | —        | Conformance level (default: `ProfileMinimum`)                                                 |
| `BusinessProcessID`                                                     | all      | Business process (BT-23): Peppol process identifier, or the French cadre de facturation       |
| `TypeCode`                                                              | all      | Overrides the type code derived from the document type (`"380"` invoice, `"381"` credit note) |
| `TaxCategoryCode`                                                       | BASIC-WL | VAT category for taxes without a `Category` (default: `"S"`)                                  |
| `NoTaxCategoryCode`, `NoTaxExemptionReason`, `NoTaxExemptionReasonCode` | BASIC-WL | VAT category and mandatory reason for items without a `Tax` (default category: `"E"`)         |
| `DocumentDiscountReason`                                                | BASIC-WL | Reason text of the allowance generated from `doc.Discount` (default: `"Discount"`)            |
| `VATPointDateCode`                                                      | BASIC-WL | When VAT becomes chargeable (BT-8): `"5"` invoice date, `"29"` delivery, `"72"` payment       |
| `TaxCurrencyCode`, `TaxCurrencyRate`                                    | BASIC-WL | Currency of the tax authority (BT-6) and conversion rate; adds the VAT total in that currency |
| `ItemDefaultUnitCode`                                                   | BASIC    | Unit code for items whose `Unit` is empty (default: `"C62"`)                                  |
| `ShowIcon`                                                              | —        | Place the Factur-X profile icon in the bottom-right corner of the first page                  |
| `FrenchRules`, `FrenchFlow`                                             | —        | Enable the French e-invoicing rules, see [French e-invoicing](#french-e-invoicing)            |

---

## License

Distributed under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0).
See [LICENSE](./LICENSE) and [NOTICE](./NOTICE) for details.
