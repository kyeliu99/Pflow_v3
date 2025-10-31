"""Database models for the form service."""
from __future__ import annotations

from django.db import models


class Form(models.Model):
    """A reusable form definition."""

    name = models.CharField(max_length=255)
    description = models.TextField(blank=True)
    version = models.PositiveIntegerField(default=1)
    is_active = models.BooleanField(default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        ordering = ["name", "id"]

    def __str__(self) -> str:
        return f"{self.name} v{self.version}"


class FormField(models.Model):
    """A field that belongs to a form."""

    TEXT = "text"
    NUMBER = "number"
    DATE = "date"
    SELECT = "select"

    FIELD_TYPES = [
        (TEXT, "Text"),
        (NUMBER, "Number"),
        (DATE, "Date"),
        (SELECT, "Select"),
    ]

    form = models.ForeignKey(Form, related_name="fields", on_delete=models.CASCADE)
    name = models.CharField(max_length=255)
    label = models.CharField(max_length=255)
    field_type = models.CharField(max_length=32, choices=FIELD_TYPES)
    required = models.BooleanField(default=False)
    order = models.PositiveIntegerField(default=0)
    metadata = models.JSONField(default=dict, blank=True)

    class Meta:
        ordering = ["order", "id"]
        unique_together = ("form", "name")

    def __str__(self) -> str:
        return f"{self.label} ({self.field_type})"
