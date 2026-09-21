FROM python:3.13-slim AS test
WORKDIR /app
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
COPY pyproject.toml ./
COPY src ./src
COPY tests ./tests
RUN pip install --no-cache-dir '.[test]'
RUN pytest -q

FROM test AS runtime
EXPOSE 8082
ENTRYPOINT ["policy-registry"]

