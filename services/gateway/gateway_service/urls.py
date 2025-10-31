"""URL configuration for the gateway service."""
from django.urls import include, path

urlpatterns = [
    path("api/", include("api.urls")),
]
