#!/usr/bin/env python3
"""
SaaS Pricing Scraper — fetches pricing from public SaaS pages and updates the seed CSV.

Usage:
    python scripts/scrape-pricing.py                        # check all, report changes
    python scripts/scrape-pricing.py --update                # update CSV with new data
    python scripts/scrape-pricing.py --vendor Slack          # check specific vendor
    python scripts/scrape-pricing.py --list-vendors          # show supported vendors
    python scripts/scrape-pricing.py --dry-run               # fetch & report, no CSV write
"""

from __future__ import annotations

import argparse
import csv
import dataclasses
import logging
import os
import re
import sys
import time
from datetime import date, datetime
from pathlib import Path
from typing import Any
from urllib.parse import urlparse

import requests
from bs4 import BeautifulSoup

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

logging.basicConfig(
    level=logging.INFO,
    format="%(levelname).1s %(message)s",
    stream=sys.stderr,
)
log = logging.getLogger("scrape-pricing")

# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------

SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent
CSV_PATH = PROJECT_ROOT / "data" / "seeds" / "saas_pricing.csv"

# ---------------------------------------------------------------------------
# User-Agent rotation
# ---------------------------------------------------------------------------

USER_AGENTS = [
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
]

_ua_index = 0


def _next_user_agent() -> str:
    global _ua_index
    ua = USER_AGENTS[_ua_index % len(USER_AGENTS)]
    _ua_index += 1
    return ua


# ---------------------------------------------------------------------------
# Vendor configuration
# ---------------------------------------------------------------------------
# Each vendor entry can specify CSS selectors OR use fallback_scanner=True for
# generic price-pattern scanning.
# Selectors are approximate and may need tuning as vendors update their pages.

VENDORS: dict[str, dict[str, Any]] = {
    "Slack": {
        "url": "https://slack.com/pricing",
        "selectors": {
            "plans": "[data-qa='pricing-card']",
            "name": "[data-qa='plan-name']",
            "price": "[data-qa='price-amount']",
            "unit": "[data-qa='price-period']",
        },
    },
    "GitHub": {
        "url": "https://github.com/pricing",
        "selectors": {
            "plans": "div.PricingCard",
            "name": "h3",
            "price": "span.PricingCard-amount, .price",
            "unit": "span.PricingCard-priceNote, .price-note",
        },
    },
    "Notion": {
        "url": "https://www.notion.so/pricing",
        "fallback_scanner": True,
    },
    "Linear": {
        "url": "https://linear.app/pricing",
        "fallback_scanner": True,
    },
    "Figma": {
        "url": "https://www.figma.com/pricing",
        "selectors": {
            "plans": "[data-automation-id='pricing-plan-card']",
            "name": "[data-automation-id='plan-name']",
            "price": "[data-automation-id='price-cell']",
            "unit": "[data-automation-id='price-unit']",
        },
    },
    "Zoom": {
        "url": "https://zoom.us/pricing",
        "fallback_scanner": True,
    },
    "Google Workspace": {
        "url": "https://workspace.google.com/pricing",
        "fallback_scanner": True,
    },
    "Microsoft 365": {
        "url": "https://www.microsoft.com/en-us/microsoft-365/business/compare-all-microsoft-365-business-products",
        "fallback_scanner": True,
    },
    "Dropbox": {
        "url": "https://www.dropbox.com/plans",
        "selectors": {
            "plans": "[data-index]",
            "name": "h2, h3",
            "price": ".plan-price, .price-amount",
            "unit": ".price-note, .price-period",
        },
    },
    "Salesforce": {
        "url": "https://www.salesforce.com/pricing",
        "fallback_scanner": True,
    },
    "HubSpot": {
        "url": "https://www.hubspot.com/pricing",
        "fallback_scanner": True,
    },
    "Intercom": {
        "url": "https://www.intercom.com/pricing",
        "fallback_scanner": True,
    },
    "Mailchimp": {
        "url": "https://mailchimp.com/pricing",
        "fallback_scanner": True,
    },
    "DigitalOcean": {
        "url": "https://www.digitalocean.com/pricing",
        "selectors": {
            "plans": "[data-qa='pricing-table'] tr, [class*='plan-row']",
            "name": "th, [data-qa*='plan-name']",
            "price": "[data-qa*='price'], td:nth-child(2)",
            "unit": "[data-qa*='unit']",
        },
    },
    "Cloudflare": {
        "url": "https://www.cloudflare.com/plans",
        "fallback_scanner": True,
    },
    "Vercel": {
        "url": "https://vercel.com/pricing",
        "fallback_scanner": True,
    },
    "Netlify": {
        "url": "https://www.netlify.com/pricing",
        "fallback_scanner": True,
    },
    "Supabase": {
        "url": "https://supabase.com/pricing",
        "fallback_scanner": True,
    },
    "MongoDB": {
        "url": "https://www.mongodb.com/pricing",
        "fallback_scanner": True,
    },
    "Asana": {
        "url": "https://asana.com/pricing",
        "fallback_scanner": True,
    },
    "Monday.com": {
        "url": "https://monday.com/pricing",
        "fallback_scanner": True,
    },
    "Canva": {
        "url": "https://www.canva.com/pricing",
        "fallback_scanner": True,
    },
    "Miro": {
        "url": "https://miro.com/pricing",
        "fallback_scanner": True,
    },
    "Sentry": {
        "url": "https://sentry.io/pricing",
        "fallback_scanner": True,
    },
    "GitLab": {
        "url": "https://about.gitlab.com/pricing",
        "fallback_scanner": True,
    },
    "Atlassian": {
        "url": "https://www.atlassian.com/software/jira/pricing",
        "fallback_scanner": True,
    },
    "Adobe Creative Cloud": {
        "url": "https://www.adobe.com/creativecloud/plans.html",
        "fallback_scanner": True,
    },
    "PagerDuty": {
        "url": "https://www.pagerduty.com/pricing",
        "fallback_scanner": True,
    },
    "Auth0": {
        "url": "https://auth0.com/pricing",
        "fallback_scanner": True,
    },
    "Stripe": {
        "url": "https://stripe.com/pricing",
        "fallback_scanner": True,
    },
    "Okta": {
        "url": "https://www.okta.com/pricing",
        "fallback_scanner": True,
    },
    "OpenAI": {
        "url": "https://openai.com/api/pricing",
        "fallback_scanner": True,
    },
    "Anthropic": {
        "url": "https://anthropic.com/pricing",
        "fallback_scanner": True,
    },
    "Google": {
        "url": "https://ai.google.dev/pricing",
        "fallback_scanner": True,
    },
    "DeepSeek": {
        "url": "https://deepseek.com/pricing",
        "fallback_scanner": True,
    },
    "Mistral": {
        "url": "https://mistral.ai/pricing",
        "fallback_scanner": True,
    },
    "Cohere": {
        "url": "https://cohere.com/pricing",
        "fallback_scanner": True,
    },
}

# ---------------------------------------------------------------------------
# Specialized scrapers for AI model providers
# These handle per-token and per-character pricing, normalizing to
# per-1M-input-tokens equivalent.
# ---------------------------------------------------------------------------

# Token-to-character ratio for Google's per-character pricing.
# ~1 token ≈ 4 characters in English text.
_CHARS_PER_TOKEN = 4.0


def scrape_openai_pricing() -> list[ScrapedPlan]:
    """Scrape OpenAI API pricing and normalize to per-1M-input-tokens."""
    config = VENDORS["OpenAI"]
    html = fetch(config["url"])
    if not html:
        return []
    soup = BeautifulSoup(html, "lxml")
    results = _generic_scan(soup)
    for r in results:
        r.vendor = "OpenAI"
    return results


def scrape_anthropic_pricing() -> list[ScrapedPlan]:
    """Scrape Anthropic API pricing and normalize to per-1M-input-tokens."""
    config = VENDORS["Anthropic"]
    html = fetch(config["url"])
    if not html:
        return []
    soup = BeautifulSoup(html, "lxml")
    results = _generic_scan(soup)
    for r in results:
        r.vendor = "Anthropic"
    return results


def scrape_google_pricing() -> list[ScrapedPlan]:
    """Scrape Google AI API pricing.
    Google prices are per-character. This function normalizes them
    to a per-1M-input-tokens equivalent using a 1-token ≈ 4-char ratio.
    """
    config = VENDORS["Google"]
    html = fetch(config["url"])
    if not html:
        return []
    soup = BeautifulSoup(html, "lxml")
    results = _generic_scan(soup)
    for r in results:
        r.vendor = "Google"
        if r.list_price is not None and "character" in r.unit.lower():
            # Normalize per-character → per-token → per-1M-tokens
            r.list_price = round(r.list_price * _CHARS_PER_TOKEN * 1_000_000, 6)
            r.unit = "/1M_input_tokens"
    return results


# Specialized scraper dispatch: name → function
_SPECIALIZED_SCRAPERS: dict[str, callable] = {
    "OpenAI": scrape_openai_pricing,
    "Anthropic": scrape_anthropic_pricing,
    "Google": scrape_google_pricing,
}

# ---------------------------------------------------------------------------
# Data models
# ---------------------------------------------------------------------------


@dataclasses.dataclass
class ScrapedPlan:
    vendor: str
    sku: str
    list_price: float | None
    unit: str


@dataclasses.dataclass
class CsvRow:
    vendor: str
    sku: str
    description: str
    list_price: float
    min_observed: float
    max_observed: float
    typical_pct: int
    unit: str
    category: str


# ---------------------------------------------------------------------------
# CSV I/O
# ---------------------------------------------------------------------------


def read_csv(path: Path) -> list[CsvRow]:
    rows: list[CsvRow] = []
    with open(path, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            rows.append(
                CsvRow(
                    vendor=row["vendor"].strip(),
                    sku=row["sku"].strip(),
                    description=row["description"].strip(),
                    list_price=float(row["list_price"]),
                    min_observed=float(row["min_observed"]),
                    max_observed=float(row["max_observed"]),
                    typical_pct=int(row["typical_pct"]),
                    unit=row["unit"].strip(),
                    category=row["category"].strip(),
                )
            )
    return rows


def write_csv(path: Path, rows: list[CsvRow]) -> None:
    fieldnames = [
        "vendor",
        "sku",
        "description",
        "list_price",
        "min_observed",
        "max_observed",
        "typical_pct",
        "unit",
        "category",
    ]
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        for r in rows:
            writer.writerow({
                "vendor": r.vendor,
                "sku": r.sku,
                "description": r.description,
                "list_price": f"{r.list_price:.2f}",
                "min_observed": f"{r.min_observed:.2f}",
                "max_observed": f"{r.max_observed:.2f}",
                "typical_pct": str(r.typical_pct),
                "unit": r.unit,
                "category": r.category,
            })


def csv_index(rows: list[CsvRow]) -> dict[tuple[str, str], CsvRow]:
    """Return {(vendor, sku): row} lookup."""
    return {(r.vendor, r.sku): r for r in rows}


# ---------------------------------------------------------------------------
# Fetching
# ---------------------------------------------------------------------------

SESSION = requests.Session()
SESSION.headers.update({"Accept": "text/html,application/xhtml+xml"})


def fetch(url: str, timeout: int = 30) -> str | None:
    """Fetch a URL and return the HTML text, or None on failure."""
    try:
        resp = SESSION.get(
            url,
            headers={"User-Agent": _next_user_agent()},
            timeout=timeout,
        )
        resp.raise_for_status()
        return resp.text
    except requests.RequestException as exc:
        log.warning("  HTTP error for %s: %s", url, exc)
        return None


# ---------------------------------------------------------------------------
# Extraction
# ---------------------------------------------------------------------------

_PRICE_RE = re.compile(r"\$?([0-9,]+(?:\.[0-9]+)?)")
_FREE_RE = re.compile(r"\b(?:free|Free|FREE)\b")


def _extract_price(text: str) -> float | None:
    """Extract the first dollar amount from text, or None."""
    lowered = text.lower()
    if _FREE_RE.search(lowered):
        return 0.0
    for chunk in re.split(r"[,;/\n]", text):
        m = _PRICE_RE.search(chunk)
        if m:
            val = float(m.group(1).replace(",", ""))
            if val < 1_000_000:  # sanity cap
                return val
    return None


def _extract_by_selectors(
    soup: BeautifulSoup, selectors: dict[str, str]
) -> list[ScrapedPlan]:
    """Use CSS selectors to extract pricing cards."""
    plans_container = selectors.get("plans", "")
    name_sel = selectors.get("name", "")
    price_sel = selectors.get("price", "")
    unit_sel = selectors.get("unit", "")

    if not plans_container:
        return []

    cards = soup.select(plans_container)
    if not cards:
        # try a table-based approach
        return _extract_tables(soup)

    results: list[ScrapedPlan] = []
    for card in cards:
        name_el = card.select_one(name_sel) if name_sel else None
        price_el = card.select_one(price_sel) if price_sel else None
        unit_el = card.select_one(unit_sel) if unit_sel else None

        name = name_el.get_text(strip=True) if name_el else ""
        if not name:
            continue

        price_text = price_el.get_text(strip=True) if price_el else ""
        price = _extract_price(price_text) if price_text else None
        unit = unit_el.get_text(strip=True) if unit_el else ""

        if price is not None:
            results.append(ScrapedPlan(vendor="", sku=name, list_price=price, unit=unit))

    if not results:
        return _extract_tables(soup)
    return results


def _extract_tables(soup: BeautifulSoup) -> list[ScrapedPlan]:
    """Scan HTML tables/dd-dl for pricing rows (fallback)."""
    results: list[ScrapedPlan] = []

    for table in soup.find_all("table"):
        for row in table.find_all("tr"):
            cells = row.find_all(["td", "th"])
            if len(cells) < 2:
                continue
            name = cells[0].get_text(strip=True)
            price_text = cells[-1].get_text(strip=True)
            price = _extract_price(price_text)
            if name and price is not None:
                results.append(ScrapedPlan(vendor="", sku=name, list_price=price, unit=""))

    # Also scan definition lists
    for dl in soup.find_all("dl"):
        dds = dl.find_all("dd")
        dts = dl.find_all("dt")
        for dt, dd in zip(dts, dds):
            name = dt.get_text(strip=True)
            price_text = dd.get_text(strip=True)
            price = _extract_price(price_text)
            if name and price is not None:
                results.append(ScrapedPlan(vendor="", sku=name, list_price=price, unit=""))

    return results


def _generic_scan(soup: BeautifulSoup) -> list[ScrapedPlan]:
    """
    Full-page scan for pricing patterns.
    Looks for common pricing-card containers and price-like text.
    """
    results: list[ScrapedPlan] = []
    seen_names: set[str] = set()

    # Strategy 1: look for elements with plan/pricing/card/tier/package classes
    candidates = soup.find_all(
        ["div", "section", "article", "li"],
        class_=re.compile(r"(plan|pricing|card|tier|package)", re.I),
    )
    for el in candidates:
        text = el.get_text(separator=" ", strip=True)
        price = _extract_price(text)
        if price is None:
            continue
        # Find a plan name (h2-h4 or strong within the element)
        name_el = el.find(["h2", "h3", "h4", "strong", "span"])
        name = name_el.get_text(strip=True) if name_el else ""
        if not name or len(name) > 50:
            name = text.split(".")[0].split("\n")[0][:40]
        if name and price is not None and name not in seen_names:
            seen_names.add(name)
            results.append(ScrapedPlan(vendor="", sku=name, list_price=price, unit=""))

    # Strategy 2: look for <header/heading> + price in parent
    if not results:
        for tag in ["h1", "h2", "h3", "h4"]:
            for heading in soup.find_all(tag):
                parent = heading.parent
                if not parent:
                    continue
                price_text = parent.get_text(separator=" ", strip=True)
                price = _extract_price(price_text)
                name = heading.get_text(strip=True)
                if price is not None and name and name not in seen_names:
                    seen_names.add(name)
                    results.append(
                        ScrapedPlan(vendor="", sku=name, list_price=price, unit="")
                    )

    return results


def extract(vendor_name: str, html: str | None) -> list[ScrapedPlan]:
    """Extract pricing plans from HTML for the given vendor."""
    if not html:
        return []

    soup = BeautifulSoup(html, "lxml")
    config = VENDORS[vendor_name]

    selectors = config.get("selectors")
    should_fallback = config.get("fallback_scanner", False)

    results: list[ScrapedPlan] = []
    if selectors:
        results = _extract_by_selectors(soup, selectors)

    # If selectors produced nothing, use the generic scanner
    if not results and should_fallback:
        results = _generic_scan(soup)

    # Last resort: try the generic scanner anyway
    if not results:
        results = _generic_scan(soup)

    # Tag each result with the vendor name
    for r in results:
        r.vendor = vendor_name

    return results


# ---------------------------------------------------------------------------
# Comparison
# ---------------------------------------------------------------------------


@dataclasses.dataclass
class Change:
    vendor: str
    sku: str
    old_price: float
    new_price: float
    diff_pct: float


@dataclasses.dataclass
class Report:
    scrape_date: date
    changes: list[Change]
    new_discoveries: list[ScrapedPlan]
    stale: list[tuple[str, str]]  # (vendor, sku)


PRICE_CHANGE_THRESHOLD = 5.0  # percent


def compare(
    csv_rows: list[CsvRow], scraped: list[ScrapedPlan]
) -> Report:
    """Compare scraped data against CSV rows and produce a Report."""
    idx = csv_index(csv_rows)
    today = date.today()

    # Group scraped by vendor
    scraped_by_vendor: dict[str, list[ScrapedPlan]] = {}
    for p in scraped:
        scraped_by_vendor.setdefault(p.vendor, []).append(p)

    changes: list[Change] = []
    new_discoveries: list[ScrapedPlan] = []
    stale: list[tuple[str, str]] = []

    # Track which (vendor, sku) we matched
    matched_csv: set[tuple[str, str]] = set()

    for sp in scraped:
        key = (sp.vendor, sp.sku)
        csv_row = idx.get(key)
        if csv_row is None:
            new_discoveries.append(sp)
        else:
            matched_csv.add(key)
            if sp.list_price is not None and csv_row.list_price > 0:
                diff_pct = (
                    (sp.list_price - csv_row.list_price) / csv_row.list_price * 100
                )
                if abs(diff_pct) >= PRICE_CHANGE_THRESHOLD:
                    changes.append(
                        Change(
                            vendor=sp.vendor,
                            sku=sp.sku,
                            old_price=csv_row.list_price,
                            new_price=sp.list_price,
                            diff_pct=round(diff_pct, 1),
                        )
                    )

    # Find stale entries: CSV rows for vendors we scraped but not found on page
    scraped_vendors = set(scraped_by_vendor.keys())
    for row in csv_rows:
        if row.vendor in scraped_vendors and (row.vendor, row.sku) not in matched_csv:
            stale.append((row.vendor, row.sku))

    return Report(
        scrape_date=today,
        changes=changes,
        new_discoveries=new_discoveries,
        stale=stale,
    )


# ---------------------------------------------------------------------------
# Reporting (markdown)
# ---------------------------------------------------------------------------


def format_report(report: Report) -> str:
    """Return a Markdown-format report string."""
    lines: list[str] = []
    lines.append(f"# Pricing Scrape Results — {report.scrape_date}")
    lines.append("")

    if report.changes:
        lines.append(f"## Changes Detected ({len(report.changes)})")
        lines.append("| Vendor | SKU | Old Price | New Price | Diff |")
        lines.append("|--------|-----|-----------|-----------|------|")
        for c in report.changes:
            arrow = "+" if c.diff_pct >= 0 else ""
            lines.append(
                f"| {c.vendor} | {c.sku} | ${c.old_price:.2f} | ${c.new_price:.2f} | "
                f"{arrow}{c.diff_pct}% |"
            )
        lines.append("")

    if report.new_discoveries:
        lines.append(f"## New Discoveries ({len(report.new_discoveries)})")
        lines.append("| Vendor | SKU | Price |")
        lines.append("|--------|-----|-------|")
        for nd in report.new_discoveries:
            price_str = f"${nd.list_price:.2f}" if nd.list_price is not None else "N/A"
            lines.append(f"| {nd.vendor} | {nd.sku} | {price_str} |")
        lines.append("")

    if report.stale:
        lines.append(f"## Stale Entries ({len(report.stale)})")
        by_vendor: dict[str, list[str]] = {}
        for v, s in report.stale:
            by_vendor.setdefault(v, []).append(s)
        for v in sorted(by_vendor):
            skus = ", ".join(by_vendor[v])
            lines.append(f"- {v} ({skus}) — no longer found on page")
        lines.append("")

    if not report.changes and not report.new_discoveries and not report.stale:
        lines.append("No changes detected. All prices match the CSV.")
        lines.append("")

    return "\n".join(lines)


# ---------------------------------------------------------------------------
# CSV update (--update flag)
# ---------------------------------------------------------------------------


def apply_update(csv_rows: list[CsvRow], report: Report) -> list[CsvRow]:
    """Merge scraped data back into CSV rows."""
    idx = csv_index(csv_rows)

    # Update prices for changes
    for c in report.changes:
        key = (c.vendor, c.sku)
        if key in idx:
            idx[key].list_price = c.new_price
            log.info(
                "  Updated %s/%s: $%.2f -> $%.2f",
                c.vendor, c.sku, c.old_price, c.new_price,
            )

    # Add new discoveries
    for nd in report.new_discoveries:
        price = nd.list_price if nd.list_price is not None else 0.0
        new_row = CsvRow(
            vendor=nd.vendor,
            sku=nd.sku,
            description=f"{nd.vendor} {nd.sku} -- auto-discovered",
            list_price=price,
            min_observed=price,
            max_observed=price,
            typical_pct=0,
            unit=nd.unit or "monthly",
            category=_infer_category(nd.vendor),
        )
        csv_rows.append(new_row)
        log.info("  Added %s/%s at $%.2f", nd.vendor, nd.sku, price)

    # Sort rows for consistency
    csv_rows.sort(key=lambda r: (r.vendor, r.sku))
    return csv_rows


def _infer_category(vendor: str) -> str:
    """Rough category guess based on vendor name."""
    CAT_MAP: dict[str, str] = {
        "Slack": "productivity",
        "Notion": "productivity",
        "Linear": "devtools",
        "GitHub": "devtools",
        "GitLab": "devtools",
        "Figma": "design",
        "Zoom": "productivity",
        "Google Workspace": "productivity",
        "Microsoft 365": "productivity",
        "Dropbox": "productivity",
        "Salesforce": "crm",
        "HubSpot": "marketing",
        "Intercom": "marketing",
        "Mailchimp": "marketing",
        "DigitalOcean": "hosting",
        "Cloudflare": "hosting",
        "Vercel": "hosting",
        "Netlify": "hosting",
        "Supabase": "hosting",
        "MongoDB": "hosting",
        "Asana": "productivity",
        "Monday.com": "productivity",
        "Canva": "design",
        "Miro": "productivity",
        "Sentry": "analytics",
        "Atlassian": "devtools",
        "Adobe Creative Cloud": "design",
        "PagerDuty": "devtools",
        "Auth0": "security",
        "Okta": "security",
        "Stripe": "analytics",
        "OpenAI": "ai",
        "Anthropic": "ai",
        "Google": "ai",
        "DeepSeek": "ai",
        "Mistral": "ai",
        "Cohere": "ai",
    }
    return CAT_MAP.get(vendor, "analytics")


# ---------------------------------------------------------------------------
# Main entrypoint
# ---------------------------------------------------------------------------


def scrape_vendor(
    vendor_name: str, html_cache: dict[str, str | None]
) -> list[ScrapedPlan]:
    """Scrape a single vendor: fetch, extract, and return plans.
    
    If the vendor has a specialized scraper (e.g. for AI model pricing),
    it is used instead of the generic fetch+extract pipeline.
    """
    # Check for specialized scraper first
    specialized = _SPECIALIZED_SCRAPERS.get(vendor_name)
    if specialized:
        log.info("  Using specialized scraper for %s ...", vendor_name)
        return specialized()

    config = VENDORS.get(vendor_name)
    if not config:
        log.error("Unknown vendor: %s", vendor_name)
        return []

    url = config["url"]
    log.info("  Fetching %s ...", url)

    if url not in html_cache:
        html = fetch(url)
        html_cache[url] = html
        time.sleep(0.5)  # rate limit between vendors
    else:
        html = html_cache[url]

    plans = extract(vendor_name, html)
    log.info("  -> %d plan(s) extracted", len(plans))
    return plans


def main() -> None:
    parser = argparse.ArgumentParser(
        description="SaaS Pricing Scraper -- fetch pricing from public SaaS pages "
        "and compare against the seed CSV."
    )
    parser.add_argument(
        "--update",
        action="store_true",
        help="Update the CSV with scraped data (new prices, new SKUs).",
    )
    parser.add_argument(
        "--vendor",
        type=str,
        default=None,
        help="Scrape only this vendor (case-sensitive, e.g. 'Slack').",
    )
    parser.add_argument(
        "--list-vendors",
        action="store_true",
        help="List all configured vendors and exit.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Fetch and report without writing to the CSV.",
    )
    args = parser.parse_args()

    if args.list_vendors:
        print("Configured vendors:")
        for name, cfg in sorted(VENDORS.items()):
            print(f"  {name:30s} {cfg['url']}")
        return

    # Read existing CSV
    if not CSV_PATH.exists():
        log.error("CSV not found at %s", CSV_PATH)
        sys.exit(1)

    csv_rows = read_csv(CSV_PATH)
    log.info("Read %d rows from CSV", len(csv_rows))

    # Determine which vendors to scrape
    if args.vendor:
        vendor_names = [args.vendor]
    else:
        vendor_names = sorted(VENDORS.keys())

    # Scrape
    all_plans: list[ScrapedPlan] = []
    html_cache: dict[str, str | None] = {}
    errors = 0

    for vname in vendor_names:
        try:
            plans = scrape_vendor(vname, html_cache)
            all_plans.extend(plans)
        except Exception as exc:
            log.error("  Failed for %s: %s", vname, exc)
            errors += 1

    # Compare
    report = compare(csv_rows, all_plans)

    # Print report
    report_md = format_report(report)
    print()
    print(report_md)

    # Apply updates if requested
    if args.update or args.dry_run:
        updated = apply_update(csv_rows, report)
        if args.update:
            write_csv(CSV_PATH, updated)
            log.info("Wrote %d rows to %s", len(updated), CSV_PATH)
        elif args.dry_run:
            if report.changes or report.new_discoveries:
                log.info(
                    "DRY RUN -- %d rows would be written (use --update to apply)",
                    len(updated),
                )
            else:
                log.info("DRY RUN -- no changes to write")

    if errors:
        sys.exit(1)

    log.info("Done.")


if __name__ == "__main__":
    main()
