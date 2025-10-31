"""Serializers for identity records."""
from __future__ import annotations

from rest_framework import serializers

from .models import User


class UserSerializer(serializers.ModelSerializer):
    class Meta:
        model = User
        fields = [
            "id",
            "email",
            "display_name",
            "role",
            "is_active",
            "created_at",
            "updated_at",
        ]
