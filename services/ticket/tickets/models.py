"""Database models for the ticket service."""
from __future__ import annotations

from django.db import models


class Ticket(models.Model):
    """A workflow ticket raised from a form submission."""

    DRAFT = "draft"
    OPEN = "open"
    IN_PROGRESS = "in_progress"
    RESOLVED = "resolved"
    CLOSED = "closed"

    STATUS_CHOICES = [
        (DRAFT, "Draft"),
        (OPEN, "Open"),
        (IN_PROGRESS, "In Progress"),
        (RESOLVED, "Resolved"),
        (CLOSED, "Closed"),
    ]

    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"

    PRIORITY_CHOICES = [
        (LOW, "Low"),
        (MEDIUM, "Medium"),
        (HIGH, "High"),
    ]

    title = models.CharField(max_length=255)
    description = models.TextField(blank=True)
    status = models.CharField(max_length=32, choices=STATUS_CHOICES, default=OPEN)
    priority = models.CharField(max_length=16, choices=PRIORITY_CHOICES, default=MEDIUM)
    form_id = models.IntegerField()
    requester_id = models.IntegerField(null=True, blank=True)
    assignee_id = models.IntegerField(null=True, blank=True)
    workflow_id = models.IntegerField(null=True, blank=True)
    payload = models.JSONField(default=dict, blank=True)
    due_date = models.DateField(null=True, blank=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        ordering = ["-created_at", "id"]

    def __str__(self) -> str:
        return f"{self.title} ({self.status})"
