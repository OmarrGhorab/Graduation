from typing import Any, Optional


def success_response(data: Any = None, message: Optional[str] = None) -> dict:
    return {
        "success": True,
        "data": data,
        "error": None,
        "message": message,
    }


def error_response(code: str, message: str, data: Any = None) -> dict:
    return {
        "success": False,
        "data": data,
        "error": {
            "code": code,
            "message": message,
        },
        "message": message,
    }
