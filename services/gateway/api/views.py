"""HTTP endpoints for the API gateway."""
from __future__ import annotations

from typing import Any, Dict, Iterable

import requests
from django.conf import settings
from rest_framework import status
from rest_framework.decorators import api_view
from rest_framework.request import Request
from rest_framework.response import Response
from rest_framework.views import APIView

SERVICE_ENDPOINTS: Dict[str, str] = {
    "forms": settings.FORM_SERVICE_URL.rstrip("/") + "/api/forms/",
    "users": settings.IDENTITY_SERVICE_URL.rstrip("/") + "/api/users/",
    "tickets": settings.TICKET_SERVICE_URL.rstrip("/") + "/api/tickets/",
    "ticket_submissions": settings.TICKET_SERVICE_URL.rstrip("/") + "/api/tickets/submissions/",
    "ticket_queue_metrics": settings.TICKET_SERVICE_URL.rstrip("/") + "/api/tickets/queue-metrics/",
    "workflows": settings.WORKFLOW_SERVICE_URL.rstrip("/") + "/api/workflows/",
    "forms_health": settings.FORM_SERVICE_URL.rstrip("/") + "/api/healthz/",
    "identity_health": settings.IDENTITY_SERVICE_URL.rstrip("/") + "/api/healthz/",
    "tickets_health": settings.TICKET_SERVICE_URL.rstrip("/") + "/api/healthz/",
    "workflows_health": settings.WORKFLOW_SERVICE_URL.rstrip("/") + "/api/healthz/",
}


def _forward_request(method: str, base_url: str, request: Request, suffix: str = "") -> Response:
    url = base_url + suffix
    headers = {key: value for key, value in request.headers.items() if key.lower().startswith("x-")}
    if request.content_type:
        headers["Content-Type"] = request.content_type
    json_payload = None
    if method.upper() in {"POST", "PUT", "PATCH"}:
        json_payload = request.data

    try:
        response = requests.request(
            method,
            url,
            params=request.query_params,
            json=json_payload,
            headers=headers,
            timeout=settings.SERVICE_TIMEOUT,
        )
    except requests.RequestException as exc:  # pragma: no cover - network errors
        return Response(
            {"detail": f"Upstream request failed: {exc}"},
            status=status.HTTP_502_BAD_GATEWAY,
        )

    content_type = response.headers.get("Content-Type", "application/json")
    if "application/json" in content_type:
        data: Any
        try:
            data = response.json()
        except ValueError:
            data = response.text
    else:
        data = response.text

    return Response(data, status=response.status_code)


class CollectionProxyView(APIView):
    service_key: str

    def get(self, request: Request) -> Response:
        return _forward_request("GET", SERVICE_ENDPOINTS[self.service_key], request)

    def post(self, request: Request) -> Response:
        return _forward_request("POST", SERVICE_ENDPOINTS[self.service_key], request)


class DetailProxyView(APIView):
    service_key: str

    def get(self, request: Request, resource_id: int) -> Response:
        return _forward_request("GET", SERVICE_ENDPOINTS[self.service_key], request, f"{resource_id}/")

    def put(self, request: Request, resource_id: int) -> Response:
        return _forward_request("PUT", SERVICE_ENDPOINTS[self.service_key], request, f"{resource_id}/")

    def patch(self, request: Request, resource_id: int) -> Response:
        return _forward_request("PATCH", SERVICE_ENDPOINTS[self.service_key], request, f"{resource_id}/")

    def delete(self, request: Request, resource_id: int) -> Response:
        return _forward_request("DELETE", SERVICE_ENDPOINTS[self.service_key], request, f"{resource_id}/")


class FormCollectionView(CollectionProxyView):
    service_key = "forms"


class FormDetailView(DetailProxyView):
    service_key = "forms"


class UserCollectionView(CollectionProxyView):
    service_key = "users"


class UserDetailView(DetailProxyView):
    service_key = "users"


class TicketCollectionView(CollectionProxyView):
    service_key = "tickets"


class TicketDetailView(DetailProxyView):
    service_key = "tickets"


class TicketSubmissionCollectionView(APIView):
    def post(self, request: Request) -> Response:
        return _forward_request("POST", SERVICE_ENDPOINTS["ticket_submissions"], request)


class TicketSubmissionDetailView(APIView):
    def get(self, request: Request, submission_id: str) -> Response:
        return _forward_request(
            "GET",
            SERVICE_ENDPOINTS["ticket_submissions"],
            request,
            f"{submission_id}/",
        )


class WorkflowCollectionView(CollectionProxyView):
    service_key = "workflows"


class WorkflowDetailView(DetailProxyView):
    service_key = "workflows"


class TicketQueueMetricsView(APIView):
    def get(self, request: Request) -> Response:
        return _forward_request(
            "GET", SERVICE_ENDPOINTS["ticket_queue_metrics"], request
        )


@api_view(["GET"])
def health(_: Request) -> Response:
    """Expose a ready signal for load balancers."""

    return Response({"status": "ok"})


@api_view(["GET"])
def overview(_: Request) -> Response:
    """Provide the status of each microservice."""

    def _fetch_collection(url: str) -> Iterable[Dict[str, Any]]:
        try:
            response = requests.get(url, timeout=settings.SERVICE_TIMEOUT)
            if response.status_code != 200:
                return []
            payload = response.json()
        except (requests.RequestException, ValueError):  # pragma: no cover - network errors
            return []

        if isinstance(payload, dict) and "results" in payload:
            items = payload["results"]
        else:
            items = payload
        return items if isinstance(items, list) else []

    forms = list(_fetch_collection(SERVICE_ENDPOINTS["forms"]))
    tickets = list(_fetch_collection(SERVICE_ENDPOINTS["tickets"]))
    users = list(_fetch_collection(SERVICE_ENDPOINTS["users"]))
    workflows = list(_fetch_collection(SERVICE_ENDPOINTS["workflows"]))

    try:
        queue_response = requests.get(
            SERVICE_ENDPOINTS["ticket_queue_metrics"],
            timeout=settings.SERVICE_TIMEOUT,
        )
        queue_response.raise_for_status()
        queue_metrics = queue_response.json()
    except (requests.RequestException, ValueError):  # pragma: no cover - network errors
        queue_metrics = {
            "pending": 0,
            "processing": 0,
            "completed": 0,
            "failed": 0,
            "oldestPendingSeconds": 0,
        }

    ticket_status_counts: Dict[str, int] = {}
    for ticket in tickets:
        status_value = str(ticket.get("status", "unknown"))
        ticket_status_counts[status_value] = ticket_status_counts.get(status_value, 0) + 1

    published_workflows = sum(1 for workflow in workflows if workflow.get("is_active"))

    return Response(
        {
            "forms": {"total": len(forms)},
            "tickets": {
                "total": len(tickets),
                "byStatus": ticket_status_counts,
                "queue": queue_metrics,
            },
            "users": {"total": len(users)},
            "workflows": {"total": len(workflows), "published": published_workflows},
        }
    )
