# Collected Auth

## Set Up

### 1. Create a new project at Google Cloud

### 2. Install `gcloud`, sign in

## Development

### 1. Create a **.envrc** file:

```
export PROJECT=YOUR_GOOGLE_CLOUD_PROJECT_ID
```

### 2. Create a **.env** file:

```
GITHUB_CLIENT_ID = …
GITHUB_CLIENT_SECRET = …
GITHUB_REDIRECT_URL = "http://localhost:8080/signin/github/callback"
```

### 3. Run `make dev`

## Deploying

### 1. Copy **app.yaml** to a **app.prod.yaml** file, and add:

```yaml
env_variables:
  GITHUB_CLIENT_ID: "…"
  GITHUB_CLIENT_SECRET: "…"
  GITHUB_REDIRECT_URL: "…/signin/github/callback"
```

### 2. Run `make deploy`