[![ci](https://github.com/angelodlfrtr/go-invoice-generator/actions/workflows/ci.yml/badge.svg)](https://github.com/angelodlfrtr/go-invoice-generator/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/angelodlfrtr/go-invoice-generator)](https://goreportcard.com/report/github.com/angelodlfrtr/go-invoice-generator)
[![GoDoc](https://godoc.org/github.com/angelodlfrtr/go-invoice-generator?status.svg)](https://godoc.org/github.com/angelodlfrtr/go-invoice-generator)

# go-invoice-generator

A Go package for generating **invoices**, **delivery notes**, and **quotations** as PDF files,
built on top of [go-pdf/fpdf](https://codeberg.org/go-pdf/fpdf).

## Features

- Three document types: Invoice, Quotation, Delivery Note
- Per-item tax and discount (percentage or fixed amount)
- Named taxes with per-name breakdown in the totals block
- Document-level discount applied after item discounts
- Default tax applied automatically to items that have none
- Programmatic access to all totals (no need to build the PDF first)
- Custom header and footer with optional pagination
- Unicode support via a configurable translation function
- Fully customisable labels, colours, and currency formatting
- Output to file or `[]byte`
- Roboto font embedded by default — no external font files required
- WIP / Experimental: Optional [Factur-X](#factur-x--wip--experimental) subpackage produces **PDF/A-3B** compliant e-invoices for all five profiles, verified with veraPDF and mustangproject

---

## v0.7.0 — Breaking Changes

The generator code has moved to the `generator` subpackage. Update your imports:

```diff
-import generator "github.com/angelodlfrtr/go-invoice-generator"
+import generator "github.com/angelodlfrtr/go-invoice-generator/generator"
```

The module path (`github.com/angelodlfrtr/go-invoice-generator`) is unchanged.

---

## Installation

```sh
go get github.com/angelodlfrtr/go-invoice-generator/generator
```

## Quick start

```go
package main

import (
	"log"
	"os"

	generator "github.com/angelodlfrtr/go-invoice-generator/generator"
)

func main() {
	doc, err := generator.New(generator.Invoice, &generator.Options{
		TextTypeInvoice: "INVOICE",
		CurrencySymbol:  "$ ",
	})
	if err != nil {
		log.Fatal(err)
	}

	doc.SetRef("INV-2024-001")
	doc.SetDate("01/01/2024")
	doc.SetPaymentTerm("01/02/2024")

	doc.SetCompany(&generator.Contact{
		Name: "Acme Corp",
		Address: &generator.Address{
			Address:    "1 Market Street",
			PostalCode: "94105",
			City:       "San Francisco",
			Country:    "USA",
		},
	})

	doc.SetCustomer(&generator.Contact{
		Name: "John Doe",
		Address: &generator.Address{
			Address:    "42 Main Street",
			PostalCode: "10001",
			City:       "New York",
			Country:    "USA",
		},
	})

	doc.AppendItem(&generator.Item{
		Name:     "Consulting",
		UnitCost: "150.00",
		Quantity: "8",
		Tax:      &generator.Tax{Percent: "20"},
	})

	pdf, err := doc.Build()
	if err != nil {
		log.Fatal(err)
	}

	if err := pdf.OutputFileAndClose("invoice.pdf"); err != nil {
		log.Fatal(err)
	}
}
```

## Example output

![DeliveryNoteExample](support/example.png)

---

## Document types

| Constant                 | Value             |
| ------------------------ | ----------------- |
| `generator.Invoice`      | `"INVOICE"`       |
| `generator.Quotation`    | `"QUOTATION"`     |
| `generator.DeliveryNote` | `"DELIVERY_NOTE"` |

```go
doc, err := generator.New(generator.Quotation, &generator.Options{})
```

---

## Options

All fields are optional and have sensible defaults.

```go
doc, err := generator.New(generator.Invoice, &generator.Options{
	// Currency formatting
	CurrencySymbol:    "€ ",  // default: "€ "
	CurrencyPrecision: 2,     // default: 2
	CurrencyDecimal:   ".",   // default: "."
	CurrencyThousand:  " ",   // default: " "

	// Localised labels
	TextTypeInvoice:        "INVOICE",
	TextTypeQuotation:      "QUOTATION",
	TextTypeDeliveryNote:   "DELIVERY NOTE",
	TextRefTitle:           "Ref.",
	TextVersionTitle:       "Version",
	TextDateTitle:          "Date",
	TextPaymentTermTitle:   "Payment term",
	TextItemsNameTitle:     "Name",
	TextItemsUnitCostTitle: "Unit price",
	TextItemsQuantityTitle: "Qty",
	TextItemsTotalHTTitle:  "Total no tax",
	TextItemsTaxTitle:      "Tax",
	TextItemsDiscountTitle: "Discount",
	TextItemsTotalTTCTitle: "Total",
	TextTotalTotal:         "Total",
	TextTotalDiscounted:    "Total discounted",
	TextTotalTax:           "Tax",
	TextTotalTaxOther:      "Other",  // label for unnamed taxes in breakdown (default: "Other")
	TextTotalWithTax:       "Total with tax",

	// Colours (RGB)
	BaseTextColor: []int{35, 35, 35},
	GreyTextColor: []int{82, 82, 82},
	GreyBgColor:   []int{232, 232, 232},
	DarkBgColor:   []int{212, 212, 212},

	// Font family name. Roboto is embedded and used by default; any font
	// registered on the underlying fpdf instance can be used here.
	Font:     "Roboto",  // default: "Roboto"
	BoldFont: "Roboto",  // default: "Roboto"
})
```

---

## Contacts

Both the company and the customer are `Contact` values. A logo can be embedded
as a `[]byte` (PNG or JPEG).

```go
logoBytes, err := os.ReadFile("logo.png")
if err != nil {
	log.Fatal(err)
}

doc.SetCompany(&generator.Contact{
	Name: "Acme Corp",
	Logo: logoBytes,
	Address: &generator.Address{
		Address:    "1 Market Street",
		Address2:   "Suite 200",          // optional second line
		PostalCode: "94105",
		City:       "San Francisco",
		Country:    "USA",
	},
	AddtionnalInfo: []string{         // extra lines printed below the address
		"VAT: FR12345678901",
		"SIRET: 123 456 789 00010",
	},
})

doc.SetCustomer(&generator.Contact{
	Name: "John Doe",
	Address: &generator.Address{
		Address:    "42 Main Street",
		PostalCode: "10001",
		City:       "New York",
		Country:    "USA",
	},
})
```

---

## Items

Each item has a name, optional description, unit cost, quantity, and optional
tax and discount.

```go
doc.AppendItem(&generator.Item{
	Name:        "Web development",
	Description: "Frontend and backend implementation",
	UnitCost:    "1200.00",
	Quantity:    "3",
	Tax:         &generator.Tax{Percent: "20"},
	Discount:    &generator.Discount{Percent: "10"},
})
```

`UnitCost` and `Quantity` are strings to avoid floating-point precision issues;
the library uses [shopspring/decimal](https://github.com/shopspring/decimal) internally.

### Tax

A tax is either a **percentage** or a **fixed amount** — not both. An optional `Name`
groups taxes in the totals breakdown.

```go
// 20% VAT
Tax: &generator.Tax{Percent: "20", Name: "VAT 20%"}

// Reduced 5.5% VAT
Tax: &generator.Tax{Percent: "5.5", Name: "VAT 5.5%"}

// Fixed €89 eco-tax regardless of quantity
Tax: &generator.Tax{Amount: "89", Name: "Eco tax"}

// Unnamed — grouped under "Other" in the breakdown
Tax: &generator.Tax{Percent: "10"}
```

When two or more distinct tax names are present the totals block shows the overall
tax amount (same as before) followed by a smaller per-name breakdown. Taxes without a
name are grouped under the label set by `Options.TextTotalTaxOther` (default: `"Other"`).

For e-invoicing exports (see [Factur-X](#factur-x--wip--experimental)) a tax can also
carry its EN 16931 VAT category and, when the category requires it, an exemption reason.
These fields do not change the rendered PDF.

```go
// Exempt line: category E needs a reason or a VATEX reason code
Tax: &generator.Tax{
	Percent:             "0",
	Category:            "E",
	ExemptionReason:     "TVA non applicable, art. 293 B du CGI",
	ExemptionReasonCode: "VATEX-FR-FRANCHISE",
}

// Intra-community supply (reverse charge by the buyer)
Tax: &generator.Tax{Percent: "0", Category: "K", ExemptionReason: "Intra-community supply"}
```

### Discount

A discount is either a **percentage** or a **fixed amount** — not both.

```go
// 30% off the item subtotal
Discount: &generator.Discount{Percent: "30"}

// Fixed €50 deducted from the item subtotal
Discount: &generator.Discount{Amount: "50"}
```

---

## Default tax

`SetDefaultTax` applies a tax to every item that does not have its own `Tax` field.

```go
doc.SetDefaultTax(&generator.Tax{Percent: "20"})
```

Items that already have a `Tax` are not affected.

---

## Document-level discount

A document discount is applied to the subtotal after all item discounts. It
reduces both the pre-tax total and (proportionally) the tax due.

```go
// Fixed amount discount
doc.SetDiscount(&generator.Discount{Amount: "500"})

// Percentage discount
doc.SetDiscount(&generator.Discount{Percent: "5"})
```

---

## Totals

All totals are available programmatically after calling `Build()` (which runs
`Validate()` internally). You can also call `Validate()` directly if you only
need the numbers without generating a PDF.

```go
if err := doc.Validate(); err != nil {
	log.Fatal(err)
}

fmt.Println(doc.TotalWithoutTaxAndWithoutDocumentDiscount()) // sum of item subtotals after item discounts
fmt.Println(doc.DocumentDiscount())                         // document discount amount
fmt.Println(doc.TotalWithoutTax())                          // above minus document discount
fmt.Println(doc.Tax())                                      // total tax (respects document discount)
fmt.Println(doc.TotalWithTax())                             // final amount due
```

### Rounding model

Totals follow the EN 16931 model used by accounting software and e-invoicing
validators. Amounts are rounded to `Options.CurrencyPrecision` at three points:

1. each item's net amount after its own discount;
2. the document discount, which is then split across tax groups in proportion to
   their line totals (the rounding residual goes to the largest group);
3. each tax group's tax, computed on the group's rounded taxable amount.

A tax group is the set of items sharing a `Tax.Category` and rate; fixed-amount
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

Item-level helpers are also available:

```go
item := &generator.Item{UnitCost: "100", Quantity: "2", Discount: &generator.Discount{Percent: "10"}}
if err := item.Prepare(); err != nil {
	log.Fatal(err)
}

fmt.Println(item.TotalWithoutTaxAndWithoutDiscount()) // 200.00
fmt.Println(item.TotalWithoutTaxAndWithDiscount())    // 180.00
fmt.Println(item.TaxWithTotalDiscounted())            // 0 (no tax set)
fmt.Println(item.TotalWithTaxAndDiscount())           // 180.00
```

---

## Header and footer

```go
doc.SetHeader(&generator.HeaderFooter{
	Text:       "<center>Acme Corp — Confidential</center>",
	FontSize:   7,
	Pagination: true, // show "Page X/{nb}" in the top-right corner
})

doc.SetFooter(&generator.HeaderFooter{
	Text:       "<center>Acme Corp · 1 Market Street · San Francisco</center>",
	Pagination: true,
})
```

`Text` supports basic HTML tags (`<b>`, `<i>`, `<center>`).

For full control you can provide a custom function directly on the underlying
fpdf instance:

```go
hf := &generator.HeaderFooter{UseCustomFunc: true}
hf.ApplyFunc(doc.Pdf(), func() {
	// use doc.Pdf() (a *fpdf.Fpdf) to draw anything you want
})
doc.SetHeader(hf)
```

---

## Unicode support

By default the document uses the `UnicodeTranslatorFromDescriptor("")` translator
bundled with fpdf.
To use a different encoding (e.g. ISO-8859-2), check `examples/iso_8859_2_cp`.

---

## Output

**To a file:**

```go
pdf, err := doc.Build()
if err != nil {
	log.Fatal(err)
}
if err := pdf.OutputFileAndClose("out.pdf"); err != nil {
	log.Fatal(err)
}
```

**To a byte slice (e.g. for an HTTP response or S3 upload):**

```go
import "bytes"

pdf, err := doc.Build()
if err != nil {
	log.Fatal(err)
}

var buf bytes.Buffer
if err := pdf.Output(&buf); err != nil {
	log.Fatal(err)
}
// buf.Bytes() contains the PDF
```

---

## Factur-X — WIP / Experimental

The `facturx` subpackage embeds a [Factur-X](https://fnfe-mpe.org/factur-x/) (also known as ZUGFeRD 2.x) compliant CII XML into the PDF produced by `Build()`. All five conformance levels are supported and validated with mustang-cli (profile XSDs and EN 16931 schematron) and veraPDF (PDF/A-3B). In addition to the XML attachment, `Attach` automatically:

- Sets `/AFRelationship /Alternative` on the embedded file, as required by PDF/A-3.
- Inserts an sRGB ICC OutputIntent into the PDF catalog (pure Go, no external dependencies).
- Merges the required Factur-X `pdfaid`, `pdfaExtension`, and `fx:` XMP declarations into the PDF's existing XMP packet without discarding fields written by fpdf (Producer, CreationDate, etc.).

```sh
go get github.com/angelodlfrtr/go-invoice-generator/facturx
```

```go
import (
    "bytes"

    generator "github.com/angelodlfrtr/go-invoice-generator/generator"
    "github.com/angelodlfrtr/go-invoice-generator/facturx"
)

// 1. Build the PDF as usual.
pdf, err := doc.Build()
if err != nil {
    log.Fatal(err)
}

var buf bytes.Buffer
if err := pdf.Output(&buf); err != nil {
    log.Fatal(err)
}

// 2. Attach the Factur-X XML and bring the document into PDF/A-3b conformance.
result, err := facturx.Attach(buf.Bytes(), doc, facturx.Options{
    Profile:               facturx.ProfileEN16931,
    SellerTaxID:           "FR12345678901",
    SellerLegalID:         "123456789",   // SIREN
    SellerLegalIDSchemeID: "0002",        // SIRENE register
    SellerCountryCode:     "FR",
    BuyerCountryCode:      "US",
    CurrencyCode:          "EUR",
    PaymentDueDate:        "2024-02-01",
    PaymentIBAN:           "FR7630006000011234567890189",
    PaymentBIC:            "BNPAFRPP",
})
if err != nil {
    log.Fatal(err)
}

// result contains the PDF/A-3b document with the embedded factur-x.xml attachment.
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

Options that belong to a higher profile than the one selected are silently omitted from the XML.

### How the XML is derived

- **VAT breakdown.** Items are grouped by VAT category and rate (BR-CO-18). The category comes from `Tax.Category`, falling back to `TaxCategoryCode` (default `"S"`). Items without a `Tax` use `NoTaxCategoryCode` (default `"E"`, exempt) and must be given a reason through `NoTaxExemptionReason` or `NoTaxExemptionReasonCode`.
- **Document discount.** `doc.Discount` becomes one document-level allowance per VAT group (reason code `95`), split in proportion to each group's line total.
- **Totals.** Every amount comes from `doc.Breakdown()`, the same figures printed on the PDF (see [Rounding model](#rounding-model)), so the XML matches the PDF to the cent and the EN 16931 arithmetic rules hold exactly (BR-CO-10, 11, 13, 14, 15). Amounts carry two decimals, so `Options.CurrencyPrecision` above 2 is rejected.
- **Fixed-amount taxes** (`Tax.Amount`) have no VAT representation. They are accepted for MINIMUM, which carries no breakdown, and rejected with `ErrFixedAmountTax` above it; model such charges as separate items instead.
- **Dates** accept `DD/MM/YYYY`, `YYYY-MM-DD` or `YYYYMMDD`. When `PaymentDueDate` is empty and `doc.PaymentTerm` parses as a date, it is used as the due date.

### Validation

`Attach` and `BuildXML` reject documents that cannot produce a valid invoice instead of emitting XML that validators will refuse. Errors wrap one of the sentinels below; test them with `errors.Is`.

| Error                       | Raised when                                                                              |
| --------------------------- | ---------------------------------------------------------------------------------------- |
| `ErrMissingSellerID`        | Neither `SellerTaxID` nor `SellerLegalID` is set (BR-CO-26)                              |
| `ErrMissingCountry`         | Seller country missing, or buyer country missing above MINIMUM                           |
| `ErrInvalidCountryCode`     | A country code is not two letters (ISO 3166-1 alpha-2)                                   |
| `ErrInvalidCurrencyCode`    | The currency code is not three letters (ISO 4217)                                        |
| `ErrInvalidDate`            | A date does not parse                                                                    |
| `ErrInvalidTaxCategory`     | Unknown category, `S` with a 0% rate, or an exempt category with a non-zero rate         |
| `ErrMissingExemptionReason` | A category other than `S` or `Z` has no reason and no reason code                        |
| `ErrFixedAmountTax`         | An item has a fixed-amount tax and the profile is above MINIMUM                          |
| `ErrMissingPaymentTerms`    | Above MINIMUM, an amount is due but there is no due date and no payment terms (BR-CO-25) |
| `ErrInvalidPeriod`          | Only one of the invoicing period dates is set                                            |

### Options

| Field                                                                   | Profile  | Description                                                                                    |
| ----------------------------------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------- |
| `Profile`                                                               | —        | Conformance level (default: `ProfileMinimum`)                                                  |
| `CurrencyCode`                                                          | all      | ISO 4217 code (default: `"EUR"`)                                                               |
| `TypeCode`                                                              | all      | UN/CEFACT type code (default: `"380"` invoice; `"381"` credit note)                            |
| `BusinessProcessID`                                                     | all      | Business process (BT-23), e.g. the Peppol billing process identifier                           |
| `BuyerReference`                                                        | all      | Buyer's routing reference (BT-10), e.g. cost centre or Leitweg-ID                              |
| `PurchaseOrderReference`                                                | all      | Purchase order number (BT-13)                                                                  |
| `SellerTaxID`                                                           | all      | Seller VAT registration number (e.g. `"FR12345678901"`)                                        |
| `SellerLegalID`, `SellerLegalIDSchemeID`                                | all      | Seller legal registration ID (BT-30), e.g. SIREN, with its ISO 6523 scheme (`"0002"` = SIRENE) |
| `SellerCountryCode`                                                     | all      | ISO 3166-1 alpha-2 seller country; falls back to the company address country                   |
| `SellerElectronicAddress`, `SellerElectronicAddressSchemeID`            | BASIC-WL | Seller electronic address (BT-34); scheme defaults to `"EM"` (e-mail)                          |
| `SellerContactName`, `SellerContactPhone`, `SellerContactEmail`         | EN 16931 | Seller contact (BG-6)                                                                          |
| `BuyerTaxID`                                                            | BASIC-WL | Buyer VAT registration number                                                                  |
| `BuyerLegalID`, `BuyerLegalIDSchemeID`                                  | all      | Buyer legal registration ID (BT-47)                                                            |
| `BuyerCountryCode`                                                      | all      | ISO 3166-1 alpha-2 buyer country; falls back to the customer address country                   |
| `BuyerElectronicAddress`, `BuyerElectronicAddressSchemeID`              | BASIC-WL | Buyer electronic address (BT-49)                                                               |
| `DeliveryDate`                                                          | BASIC-WL | Actual delivery date (BT-72)                                                                   |
| `InvoicingPeriodStart`, `InvoicingPeriodEnd`                            | BASIC-WL | Invoicing period (BG-14); both or neither                                                      |
| `PaymentDueDate`                                                        | BASIC-WL | Payment due date; defaults to `doc.PaymentTerm` when that is a date                            |
| `PaymentTermsDescription`                                               | BASIC-WL | Free-text payment terms (BT-20)                                                                |
| `PaymentReference`                                                      | BASIC-WL | Remittance information (BT-83)                                                                 |
| `PaymentIBAN`                                                           | BASIC-WL | Seller IBAN for bank transfer                                                                  |
| `PaymentBIC`                                                            | EN 16931 | Seller BIC/SWIFT code                                                                          |
| `PaymentMeansCode`                                                      | BASIC-WL | UN/ECE 4461 payment means code (default: `"58"` when IBAN is set)                              |
| `Notes`                                                                 | BASIC-WL | Document notes (BT-22); defaults to `doc.Notes`                                                |
| `TaxCategoryCode`                                                       | BASIC-WL | VAT category for taxes without a `Category` (default: `"S"`)                                   |
| `NoTaxCategoryCode`, `NoTaxExemptionReason`, `NoTaxExemptionReasonCode` | BASIC-WL | VAT category and mandatory reason for items without a `Tax` (default category: `"E"`)          |
| `DocumentDiscountReason`                                                | BASIC-WL | Reason text of the allowance generated from `doc.Discount` (default: `"Discount"`)             |
| `ItemDefaultUnitCode`                                                   | BASIC    | UN/ECE Rec 20 unit code for all line items (default: `"C62"` piece/unit)                       |
| `ShowIcon`                                                              | —        | Place the Factur-X profile icon in the bottom-right corner of the first page                   |

---

## License

Distributed under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0).
See [LICENSE](./LICENSE) and [NOTICE](./NOTICE) for details.
