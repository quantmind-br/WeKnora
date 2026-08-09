import contextlib
import logging
import time
import uuid
from contextvars import ContextVar
from logging import LogRecord
from typing import Optional

# Configure logging
logger = logging.getLogger(__name__)

# Define context variables
request_id_var = ContextVar("request_id", default=None)
_request_start_time_ctx = ContextVar("request_start_time", default=None)


def set_request_id(request_id: str) -> None:
    """Set the request ID for the current context"""
    request_id_var.set(request_id)


def get_request_id() -> Optional[str]:
    """Get the request ID for the current context"""
    return request_id_var.get()


class MillisecondFormatter(logging.Formatter):
    """Custom log formatter that shows millisecond-level timestamps (3 digits) instead of microseconds (6 digits)"""

    def formatTime(self, record, datefmt=None):
        """Override the formatTime method to format microseconds as milliseconds"""
        # First get the standard formatted time
        result = super().formatTime(record, datefmt)

        # If a format containing .%f is used, truncate the microseconds (6 digits) to milliseconds (3 digits)
        if datefmt and ".%f" in datefmt:
            # The formatted time string should have 6 digits of microseconds at the end
            parts = result.split(".")
            if len(parts) > 1 and len(parts[1]) >= 6:
                # Keep only the first 3 digits as milliseconds
                millis = parts[1][:3]
                result = f"{parts[0]}.{millis}"

        return result


def init_logging_request_id():
    """
    Initialize logging to include request ID in log messages.
    Add the custom filter to all existing handlers
    """
    logger.info("Initializing request ID logging")
    root_logger = logging.getLogger()

    # Add the custom filter to all handlers
    for handler in root_logger.handlers:
        # Add the request ID filter
        handler.addFilter(RequestIdFilter())

        # Update the formatter to include the request ID, adjusting the format to be more compact and tidy
        formatter = logging.Formatter(
            fmt="%(asctime)s.%(msecs)03d [%(request_id)s] %(levelname)-5s %(name)-20s | %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S",
        )
        handler.setFormatter(formatter)

    logger.info(
        f"Updated {len(root_logger.handlers)} handlers with request ID formatting"
    )

    # If there are no handlers, add a standard output handler
    if not root_logger.handlers:
        handler = logging.StreamHandler()
        formatter = logging.Formatter(
            fmt="%(asctime)s.%(msecs)03d [%(request_id)s] %(levelname)-5s %(name)-20s | %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S",
        )
        handler.setFormatter(formatter)
        handler.addFilter(RequestIdFilter())
        root_logger.addHandler(handler)
        logger.info("Added new StreamHandler with request ID formatting")


class RequestIdFilter(logging.Filter):
    """Filter that adds request ID to log messages"""

    def filter(self, record: LogRecord) -> bool:
        request_id = request_id_var.get()
        if request_id is not None:
            # Add a request ID attribute to the log record, using a short format
            if len(request_id) > 8:
                # Take the first 8 characters of the ID to ensure a tidy display
                short_id = request_id[:8]
                if "-" in request_id:
                    # Try to preserve the format, e.g. test-req-1-XXX
                    parts = request_id.split("-")
                    if len(parts) >= 3:
                        # If the format is xxx-xxx-n-randompart
                        short_id = f"{parts[0]}-{parts[1]}-{parts[2]}"
                record.request_id = short_id
            else:
                record.request_id = request_id

            # Add execution-time attribute
            start_time = _request_start_time_ctx.get()
            if start_time is not None:
                elapsed_ms = int((time.time() - start_time) * 1000)
                record.elapsed_ms = elapsed_ms
                # Add execution time to the message
                if not hasattr(record, "message_with_elapsed"):
                    record.message_with_elapsed = True
                    record.msg = f"{record.msg} (elapsed: {elapsed_ms}ms)"
        else:
            # Use a placeholder if there is no request ID
            record.request_id = "no-req-id"

        return True


@contextlib.contextmanager
def request_id_context(request_id: str = None):
    """Context manager that sets a request ID for the current context

    Args:
        request_id: the request ID to use; auto-generated if None

    Example:
        with request_id_context("req-123"):
            # All logs within this code block will include the request ID req-123
            logging.info("Processing request")
    """
    # Generate or use provided request ID
    req_id = request_id or str(uuid.uuid4())

    # Set start time and request ID
    start_time = time.time()
    req_token = request_id_var.set(req_id)
    time_token = _request_start_time_ctx.set(start_time)

    logger.info(f"Starting new request with ID: {req_id}")

    try:
        yield request_id_var.get()
    finally:
        # Log completion and reset context vars
        elapsed_ms = int((time.time() - start_time) * 1000)
        logger.info(f"Request {req_id} completed in {elapsed_ms}ms")
        request_id_var.reset(req_token)
        _request_start_time_ctx.reset(time_token)
