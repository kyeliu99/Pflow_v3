"""Serializers for ticket entities."""
from __future__ import annotations

from rest_framework import serializers

from .models import Ticket


class TicketSerializer(serializers.ModelSerializer):
    class Meta:
        model = Ticket
        fields = [
            "id",
            "title",
            "description",
            "status",
            "priority",
            "form_id",
            "requester_id",
            "assignee_id",
            "workflow_id",
            "payload",
            "due_date",
            "created_at",
            "updated_at",
        ]
