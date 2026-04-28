package api

import "github.com/gofiber/fiber/v2"

const swaggerHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <title>Conevent API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css">
  <style>body{margin:0;background:#fafafa}</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js" crossorigin></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js" crossorigin></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/openapi.json",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
      plugins: [SwaggerUIBundle.plugins.DownloadUrl],
      layout: "StandaloneLayout"
    })
  </script>
</body>
</html>`

func RegisterDocs(app *fiber.App) {
	app.Get("/openapi.json", OpenAPIJSON)
	app.Get("/openapi.yaml", OpenAPIJSON)
	app.Get("/docs", SwaggerUI)
	app.Get("/docs/*", SwaggerUI)
}

func OpenAPIJSON(c *fiber.Ctx) error {
	return c.JSON(OpenAPISpec())
}

func SwaggerUI(c *fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.SendString(swaggerHTML)
}

func OpenAPISpec() fiber.Map {
	return fiber.Map{
		"openapi": "3.1.0",
		"info": fiber.Map{
			"title":       "Conevent API",
			"description": "API HTTP para gerenciar eventos, consultar saúde da aplicação e expor documentação interativa.",
			"version":     "1.0.0",
			"contact": fiber.Map{
				"name": "Equipe Conevent",
			},
		},
		"tags": []fiber.Map{
			{"name": "health", "description": "Operações de saúde da aplicação."},
			{"name": "events", "description": "Operações de criação, consulta, atualização e remoção de eventos."},
			{"name": "docs", "description": "Documentação OpenAPI e Swagger UI geradas pela aplicação."},
		},
		"servers": []fiber.Map{
			{"url": "http://localhost:3000", "description": "Ambiente local"},
			{"url": "http://localhost:30000", "description": "Kubernetes via NodePort"},
		},
		"paths": fiber.Map{
			"/health":       healthPath(),
			"/events":       eventsPath(),
			"/events/{id}":  eventByIDPath(),
			"/openapi.json": docsPath(),
			"/docs":         swaggerPath(),
		},
		"components": fiber.Map{
			"schemas": fiber.Map{
				"Event":       eventSchema(true),
				"EventInput":  eventSchema(false),
				"Error":       errorSchema(),
				"HealthCheck": healthSchema(),
			},
		},
	}
}

func healthPath() fiber.Map {
	return fiber.Map{
		"get": fiber.Map{
			"tags":        []string{"health"},
			"summary":     "Verifica a saúde da API",
			"description": "Retorna uma resposta simples para probes de liveness/readiness. Esta rota não é rastreada pelo OpenTelemetry para evitar ruído nos traces.",
			"operationId": "healthCheck",
			"responses": fiber.Map{
				"200": jsonResponse("API disponível", "#/components/schemas/HealthCheck"),
			},
		},
	}
}

func eventsPath() fiber.Map {
	return fiber.Map{
		"get": fiber.Map{
			"tags":        []string{"events"},
			"summary":     "Lista eventos",
			"description": "Retorna todos os eventos cadastrados, ordenados do mais recente para o mais antigo.",
			"operationId": "listEvents",
			"responses": fiber.Map{
				"200": fiber.Map{
					"description": "Lista de eventos",
					"content": fiber.Map{"application/json": fiber.Map{"schema": fiber.Map{
						"type":  "array",
						"items": ref("#/components/schemas/Event"),
					}}},
				},
				"500": jsonResponse("Erro interno", "#/components/schemas/Error"),
			},
		},
		"post": fiber.Map{
			"tags":        []string{"events"},
			"summary":     "Cria evento",
			"description": "Cria um evento com datas, horários, local, orçamento e status inicial.",
			"operationId": "createEvent",
			"requestBody": requestBody("#/components/schemas/EventInput"),
			"responses": fiber.Map{
				"201": jsonResponse("Evento criado", "#/components/schemas/Event"),
				"400": jsonResponse("Dados inválidos", "#/components/schemas/Error"),
				"500": jsonResponse("Erro interno", "#/components/schemas/Error"),
			},
		},
	}
}

func eventByIDPath() fiber.Map {
	return fiber.Map{
		"parameters": []fiber.Map{eventIDParam()},
		"get": fiber.Map{
			"tags":        []string{"events"},
			"summary":     "Busca evento por ID",
			"description": "Busca um evento específico usando seu UUID.",
			"operationId": "getEvent",
			"responses":   eventResponses("Evento encontrado"),
		},
		"put": fiber.Map{
			"tags":        []string{"events"},
			"summary":     "Atualiza evento por ID",
			"description": "Substitui os dados de um evento existente. O ID no corpo é opcional, mas se informado deve ser igual ao ID da URL.",
			"operationId": "updateEvent",
			"requestBody": requestBody("#/components/schemas/EventInput"),
			"responses":   eventResponses("Evento atualizado"),
		},
		"delete": fiber.Map{
			"tags":        []string{"events"},
			"summary":     "Remove evento por ID",
			"description": "Remove permanentemente um evento pelo UUID.",
			"operationId": "deleteEvent",
			"responses": fiber.Map{
				"204": fiber.Map{"description": "Evento removido"},
				"400": jsonResponse("ID inválido", "#/components/schemas/Error"),
				"404": jsonResponse("Evento não encontrado", "#/components/schemas/Error"),
				"500": jsonResponse("Erro interno", "#/components/schemas/Error"),
			},
		},
	}
}

func docsPath() fiber.Map {
	return fiber.Map{
		"get": fiber.Map{
			"tags":        []string{"docs"},
			"summary":     "Retorna a especificação OpenAPI",
			"description": "Retorna a especificação OpenAPI 3.1 gerada programaticamente pela aplicação.",
			"operationId": "openAPIJSON",
			"responses": fiber.Map{
				"200": fiber.Map{
					"description": "Documento OpenAPI",
					"content":     fiber.Map{"application/json": fiber.Map{"schema": fiber.Map{"type": "object"}}},
				},
			},
		},
	}
}

func swaggerPath() fiber.Map {
	return fiber.Map{
		"get": fiber.Map{
			"tags":        []string{"docs"},
			"summary":     "Abre a Swagger UI",
			"description": "Serve a interface Swagger UI apontando para `/openapi.json`.",
			"operationId": "swaggerUI",
			"responses": fiber.Map{
				"200": fiber.Map{
					"description": "Página HTML da Swagger UI",
					"content":     fiber.Map{"text/html": fiber.Map{"schema": fiber.Map{"type": "string"}}},
				},
			},
		},
	}
}

func eventResponses(okDescription string) fiber.Map {
	return fiber.Map{
		"200": jsonResponse(okDescription, "#/components/schemas/Event"),
		"400": jsonResponse("Requisição inválida", "#/components/schemas/Error"),
		"404": jsonResponse("Evento não encontrado", "#/components/schemas/Error"),
		"500": jsonResponse("Erro interno", "#/components/schemas/Error"),
	}
}

func requestBody(schemaRef string) fiber.Map {
	return fiber.Map{
		"required": true,
		"content": fiber.Map{
			"application/json": fiber.Map{
				"schema": ref(schemaRef),
				"examples": fiber.Map{
					"planejamento": fiber.Map{
						"summary": "Evento em planejamento",
						"value": fiber.Map{
							"name":     "Tech Conference 2026",
							"iniDate":  "2026-10-15",
							"endDate":  "2026-10-17",
							"iniTime":  "09:00",
							"endTime":  "18:00",
							"location": "Centro de Eventos",
							"budget":   15000.50,
							"status":   "Planejamento",
						},
					},
				},
			},
		},
	}
}

func jsonResponse(description, schemaRef string) fiber.Map {
	return fiber.Map{
		"description": description,
		"content": fiber.Map{
			"application/json": fiber.Map{
				"schema": ref(schemaRef),
			},
		},
	}
}

func eventIDParam() fiber.Map {
	return fiber.Map{
		"name":        "id",
		"in":          "path",
		"required":    true,
		"description": "UUID do evento",
		"schema": fiber.Map{
			"type":    "string",
			"format":  "uuid",
			"example": "550e8400-e29b-41d4-a716-446655440000",
		},
	}
}

func eventSchema(includeReadOnly bool) fiber.Map {
	properties := fiber.Map{
		"name":     stringProperty("Nome do evento", "Tech Conference 2026"),
		"iniDate":  stringProperty("Data inicial no formato YYYY-MM-DD", "2026-10-15"),
		"endDate":  stringProperty("Data final no formato YYYY-MM-DD", "2026-10-17"),
		"iniTime":  stringProperty("Hora inicial no formato HH:MM", "09:00"),
		"endTime":  stringProperty("Hora final no formato HH:MM", "18:00"),
		"location": stringProperty("Local do evento", "Centro de Eventos"),
		"budget": fiber.Map{
			"type":        "number",
			"format":      "double",
			"description": "Orçamento do evento",
			"minimum":     0,
			"example":     15000.50,
		},
		"status": fiber.Map{
			"type":        "string",
			"description": "Status do evento",
			"enum":        []string{"Planejamento", "Confirmado", "Concluído", "Cancelado"},
			"example":     "Planejamento",
		},
	}

	if includeReadOnly {
		properties["id"] = fiber.Map{"type": "string", "format": "uuid", "readOnly": true}
		properties["createdAt"] = fiber.Map{"type": "string", "format": "date-time", "readOnly": true}
	}

	return fiber.Map{
		"type":       "object",
		"required":   []string{"name", "iniDate", "endDate", "iniTime", "endTime", "location", "status"},
		"properties": properties,
	}
}

func healthSchema() fiber.Map {
	return fiber.Map{
		"type": "object",
		"properties": fiber.Map{
			"status":  stringProperty("Status da API", "ok"),
			"message": stringProperty("Mensagem de saúde", "Conevent API is running"),
		},
	}
}

func errorSchema() fiber.Map {
	return fiber.Map{
		"type": "object",
		"properties": fiber.Map{
			"error": stringProperty("Mensagem de erro", "Invalid event data"),
		},
	}
}

func stringProperty(description, example string) fiber.Map {
	return fiber.Map{"type": "string", "description": description, "example": example}
}

func ref(schemaRef string) fiber.Map {
	return fiber.Map{"$ref": schemaRef}
}
