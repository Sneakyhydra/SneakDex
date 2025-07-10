//! Data models for parsed HTML pages.
//!
//! These models represent the structured data extracted from an HTML page, including
//! text, links, images, metadata, and more. All models implement `Serialize` to support
//! easy serialization (e.g., to JSON).

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// Represents an image (`<img>`) found on the page.
#[derive(Debug, Serialize, Deserialize)]
pub struct ImageData {
    /// The `src` attribute (URL) of the image.
    pub src: String,

    /// The `alt` attribute of the image, if present.
    pub alt: Option<String>,

    /// The `title` attribute of the image, if present.
    pub title: Option<String>,
}

/// Represents a hyperlink (`<a>`) found on the page.
#[derive(Debug, Serialize, Deserialize)]
pub struct LinkData {
    /// The `href` URL of the link.
    pub url: String,

    /// The visible text of the link.
    pub text: String,

    /// Whether the link is external to the page's domain.
    pub is_external: bool,
}

/// Represents a heading (`<h1>`, `<h2>`, etc.) found on the page.
#[derive(Debug, Serialize, Deserialize)]
pub struct Heading {
    /// Heading level (e.g., 1 for `<h1>`)
    pub level: u8,

    /// The text content of the heading.
    pub text: String,
}

/// Represents a fully-parsed HTML page and its extracted data.
#[derive(Debug, Serialize, Deserialize)]
pub struct ParsedPage {
    /// The URL of the page.
    pub url: String,

    /// The page's `<title>`.
    pub title: String,

    /// The page's meta description, if present.
    pub description: Option<String>,

    /// Cleaned and normalized text content.
    pub cleaned_text: String,

    /// A list of headings (`<h1>`, `<h2>`, etc.) found on the page.
    pub headings: Vec<Heading>,

    /// All hyperlinks (`<a>`) found on the page.
    pub links: Vec<LinkData>,

    /// All images (`<img>`) found on the page.
    pub images: Vec<ImageData>,

    /// The canonical URL of the page, if specified.
    pub canonical_url: Option<String>,

    /// Detected language of the page, if determined.
    pub language: Option<String>,

    /// Word count of the `cleaned_text`.
    pub word_count: usize,

    /// The page's meta keywords, if present.
    pub meta_keywords: Option<String>,

    /// Timestamp when this page was parsed.
    pub timestamp: DateTime<Utc>,

    /// Content type of the page.
    pub content_type: String,

    /// Character encoding of the page.
    pub encoding: String,
}
