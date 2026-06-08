from app.utils.api_response import error_response, success_response


def test_success_response_uses_platform_envelope():
    response = success_response(data={"items": []}, message="ok")

    assert response == {
        "success": True,
        "data": {"items": []},
        "error": None,
        "message": "ok",
    }


def test_error_response_uses_machine_readable_code():
    response = error_response("RECOMMENDATION_GENERATION_FAILED", "Could not generate recommendations")

    assert response["success"] is False
    assert response["data"] is None
    assert response["error"]["code"] == "RECOMMENDATION_GENERATION_FAILED"
    assert response["message"] == "Could not generate recommendations"
