"""URL configuration for the form service."""
from django.urls import include, path

urlpatterns = [
    path("api/", include("forms.urls")),
]
