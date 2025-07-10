//! Error handling for the parser service.
//!
//! Defines custom error types and provides error conversion implementations
//! for better error handling throughout the application.

use thiserror::Error;

/// Custom error types for the parser service.
#[derive(Error, Debug)]
pub enum ParserError {
    /// HTML parsing errors
    #[error("Invalid URL: {url}")]
    InvalidUrl { url: String },

    /// Kafka errors
    #[error("Kafka error: {0}")]
    KafkaError(#[from] rdkafka::error::KafkaError),

    #[error("Failed to serialize message: {0}")]
    SerializationError(#[from] serde_json::Error),

    /// General errors
    #[error("Internal error: {message}")]
    InternalError { message: String },
}

impl From<std::io::Error> for ParserError {
    fn from(err: std::io::Error) -> Self {
        ParserError::InternalError {
            message: err.to_string(),
        }
    }
}

impl From<url::ParseError> for ParserError {
    fn from(err: url::ParseError) -> Self {
        ParserError::InvalidUrl {
            url: err.to_string(),
        }
    }
}

impl From<regex::Error> for ParserError {
    fn from(err: regex::Error) -> Self {
        ParserError::InternalError {
            message: format!("Regex error: {}", err),
        }
    }
} 