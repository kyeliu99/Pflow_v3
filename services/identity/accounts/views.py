"""API views for the identity service."""
from __future__ import annotations

from rest_framework import viewsets
from rest_framework.decorators import api_view
from rest_framework.filters import OrderingFilter, SearchFilter
from rest_framework.response import Response

from .models import User
from .serializers import UserSerializer


class UserViewSet(viewsets.ModelViewSet):
    queryset = User.objects.all()
    serializer_class = UserSerializer
    filter_backends = [SearchFilter, OrderingFilter]
    search_fields = ["email", "display_name"]
    ordering_fields = ["display_name", "created_at"]
    ordering = ["display_name"]


@api_view(["GET"])
def health(request):  # type: ignore[override]
    """Readiness endpoint for orchestration tooling."""

    return Response({"status": "ok"})
