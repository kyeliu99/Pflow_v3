"""Celery application for the ticket microservice."""
from __future__ import annotations

import os

from celery import Celery

os.environ.setdefault("DJANGO_SETTINGS_MODULE", "ticket_service.settings")

app = Celery("ticket_service")
app.config_from_object("django.conf:settings", namespace="CELERY")
app.autodiscover_tasks()
