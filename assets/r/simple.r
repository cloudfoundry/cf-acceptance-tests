library(shiny)

# Define the UI
ui <- fluidPage(
  titlePanel("Hello R!")
)

# Define the server code
server <- function(input, output) {
}

# Return a Shiny app object
options(shiny.port = strtoi(Sys.getenv("PORT")), shiny.host = "0.0.0.0")
shinyApp(ui = ui, server = server)
