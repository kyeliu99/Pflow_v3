"""WSGI config for the ticket service."""
import os

from django.core.wsgi import get_wsgi_application

os.environ.setdefault("DJANGO_SETTINGS_MODULE", "ticket_service.settings")

application = get_wsgi_application()
