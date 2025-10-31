"""Settings for the ticket microservice."""
from __future__ import annotations

import os
from pathlib import Path
from typing import Dict
from urllib.parse import urlparse

BASE_DIR = Path(__file__).resolve().parent.parent

SECRET_KEY = os.environ.get("DJANGO_SECRET_KEY", "ticket-service-secret-key")
DEBUG = os.environ.get("DJANGO_DEBUG", "false").lower() in {"1", "true", "yes"}
ALLOWED_HOSTS = os.environ.get("DJANGO_ALLOWED_HOSTS", "localhost,127.0.0.1").split(",")

INSTALLED_APPS = [
    "django.contrib.admin",
    "django.contrib.auth",
    "django.contrib.contenttypes",
    "django.contrib.sessions",
    "django.contrib.messages",
    "django.contrib.staticfiles",
    "django_celery_results",
    "rest_framework",
    "corsheaders",
    "tickets",
]

MIDDLEWARE = [
    "django.middleware.security.SecurityMiddleware",
    "django.contrib.sessions.middleware.SessionMiddleware",
    "corsheaders.middleware.CorsMiddleware",
    "django.middleware.common.CommonMiddleware",
    "django.middleware.csrf.CsrfViewMiddleware",
    "django.contrib.auth.middleware.AuthenticationMiddleware",
    "django.contrib.messages.middleware.MessageMiddleware",
    "django.middleware.clickjacking.XFrameOptionsMiddleware",
]

ROOT_URLCONF = "ticket_service.urls"

TEMPLATES = [
    {
        "BACKEND": "django.template.backends.django.DjangoTemplates",
        "DIRS": [],
        "APP_DIRS": True,
        "OPTIONS": {
            "context_processors": [
                "django.template.context_processors.debug",
                "django.template.context_processors.request",
                "django.contrib.auth.context_processors.auth",
                "django.contrib.messages.context_processors.messages",
            ],
        },
    }
]

WSGI_APPLICATION = "ticket_service.wsgi.application"
ASGI_APPLICATION = "ticket_service.asgi.application"

def _database_settings() -> Dict[str, Dict[str, str]]:
    url = (
        os.environ.get("TICKET_DATABASE_URL")
        or os.environ.get("DATABASE_URL")
        or "postgresql://pflow_ticket:pflow_ticket@localhost:5434/pflow_ticket"
    )
    parsed = urlparse(url)
    if parsed.scheme in {"postgres", "postgresql"}:
        return {
            "default": {
                "ENGINE": "django.db.backends.postgresql",
                "NAME": parsed.path.lstrip("/"),
                "USER": parsed.username or "",
                "PASSWORD": parsed.password or "",
                "HOST": parsed.hostname or "localhost",
                "PORT": str(parsed.port or 5432),
            }
        }

    if parsed.scheme == "sqlite":
        db_path = parsed.path or ":memory:"
        if db_path.startswith("/"):
            name = db_path
        else:
            name = str(BASE_DIR / db_path)
        return {
            "default": {
                "ENGINE": "django.db.backends.sqlite3",
                "NAME": name,
            }
        }

    raise ValueError("Supported database URLs: postgresql:// or sqlite:///")


DATABASES = _database_settings()

AUTH_PASSWORD_VALIDATORS = [
    {
        "NAME": "django.contrib.auth.password_validation.UserAttributeSimilarityValidator",
    },
    {
        "NAME": "django.contrib.auth.password_validation.MinimumLengthValidator",
    },
    {
        "NAME": "django.contrib.auth.password_validation.CommonPasswordValidator",
    },
    {
        "NAME": "django.contrib.auth.password_validation.NumericPasswordValidator",
    },
]

LANGUAGE_CODE = "en-us"
TIME_ZONE = os.environ.get("TZ", "UTC")
USE_I18N = True
USE_TZ = True

STATIC_URL = "static/"
STATIC_ROOT = str(BASE_DIR / "staticfiles")

DEFAULT_AUTO_FIELD = "django.db.models.BigAutoField"

CORS_ALLOW_ALL_ORIGINS = True

REST_FRAMEWORK = {
    "DEFAULT_RENDERER_CLASSES": ["rest_framework.renderers.JSONRenderer"],
    "DEFAULT_PARSER_CLASSES": ["rest_framework.parsers.JSONParser"],
}


def _broker_url() -> str:
    return (
        os.environ.get("TICKET_BROKER_URL")
        or os.environ.get("CELERY_BROKER_URL")
        or "redis://localhost:6379/0"
    )


CELERY_BROKER_URL = _broker_url()
CELERY_RESULT_BACKEND = (
    os.environ.get("TICKET_RESULT_BACKEND")
    or os.environ.get("CELERY_RESULT_BACKEND")
    or CELERY_BROKER_URL
)
CELERY_TASK_DEFAULT_QUEUE = os.environ.get("TICKET_QUEUE_NAME", "ticket_submissions")
CELERY_ACCEPT_CONTENT = ["json"]
CELERY_TASK_SERIALIZER = "json"
CELERY_RESULT_SERIALIZER = "json"
CELERY_TASK_ACKS_LATE = True
CELERY_TASK_TRACK_STARTED = True
CELERY_RESULT_EXTENDED = True

if os.environ.get("CELERY_ALWAYS_EAGER", "false").lower() in {"1", "true", "yes"}:
    CELERY_TASK_ALWAYS_EAGER = True
    CELERY_TASK_EAGER_PROPAGATES = True
