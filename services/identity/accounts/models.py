"""Database models for the identity service."""
from __future__ import annotations

from django.db import models


class User(models.Model):
    """A lightweight identity record."""

    ADMIN = "admin"
    AGENT = "agent"
    REQUESTER = "requester"

    ROLE_CHOICES = [
        (ADMIN, "Administrator"),
        (AGENT, "Agent"),
        (REQUESTER, "Requester"),
    ]

    email = models.EmailField(unique=True)
    display_name = models.CharField(max_length=255)
    role = models.CharField(max_length=32, choices=ROLE_CHOICES, default=REQUESTER)
    is_active = models.BooleanField(default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        ordering = ["display_name", "email"]

    def __str__(self) -> str:
        return f"{self.display_name} <{self.email}>"
