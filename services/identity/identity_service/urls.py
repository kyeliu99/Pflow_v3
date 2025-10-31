"""URL configuration for the identity service."""
from django.urls import include, path

urlpatterns = [
    path("api/", include("accounts.urls")),
]
