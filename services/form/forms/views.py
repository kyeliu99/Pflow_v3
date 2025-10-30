"""API views for the form service."""
from __future__ import annotations

from rest_framework import viewsets
from rest_framework.decorators import api_view
from rest_framework.filters import OrderingFilter, SearchFilter
from rest_framework.response import Response

from .models import Form
from .serializers import FormSerializer


class FormViewSet(viewsets.ModelViewSet):
    queryset = Form.objects.prefetch_related("fields").all()
    serializer_class = FormSerializer
    filter_backends = [SearchFilter, OrderingFilter]
    search_fields = ["name", "description"]
    ordering_fields = ["name", "updated_at"]
    ordering = ["name"]


@api_view(["GET"])
def health(request):  # type: ignore[override]
    """Readiness endpoint for orchestration tooling."""

    return Response({"status": "ok"})
