"""ASGI config for the gateway service."""
import os

from django.core.asgi import get_asgi_application

os.environ.setdefault("DJANGO_SETTINGS_MODULE", "gateway_service.settings")

application = get_asgi_application()
