#include <gtk/gtk.h>
#include <webkit/webkit.h>
#include <string.h>

static gboolean fullscreen = FALSE;

static gboolean on_close_request(GtkWindow *window, gpointer user_data) {
    (void)window;
    (void)user_data;
    g_printerr("[repro] close-request\n");
    return FALSE;
}

static void on_destroy(GtkApplication *app) {
    g_printerr("[repro] destroy\n");
    g_application_release(G_APPLICATION(app));
}

static void on_activate(GtkApplication *app, gpointer user_data) {
    (void)user_data;
    g_printerr("[repro] activate\n");

    GtkWidget *window = gtk_application_window_new(app);
    g_application_hold(G_APPLICATION(app));
    g_signal_connect(window, "close-request", G_CALLBACK(on_close_request), NULL);
    g_signal_connect_swapped(window, "destroy", G_CALLBACK(on_destroy), app);
    gtk_window_set_title(GTK_WINDOW(window), "GTK4 WebKit Transparency Repro");
    gtk_window_set_default_size(GTK_WINDOW(window), 720, 420);
    gtk_window_set_decorated(GTK_WINDOW(window), FALSE);
    if (fullscreen) {
        gtk_window_fullscreen(GTK_WINDOW(window));
    }

    GtkCssProvider *provider = gtk_css_provider_new();
    gtk_css_provider_load_from_string(
        provider,
        "window, webview { background: transparent; }"
        ".repro-window { background: transparent; }");
    gtk_style_context_add_provider_for_display(
        gtk_widget_get_display(window),
        GTK_STYLE_PROVIDER(provider),
        GTK_STYLE_PROVIDER_PRIORITY_APPLICATION);
    gtk_widget_add_css_class(window, "repro-window");

    WebKitUserContentManager *manager = webkit_user_content_manager_new();
    GtkWidget *webview = GTK_WIDGET(g_object_new(
        WEBKIT_TYPE_WEB_VIEW,
        "user-content-manager", manager,
        NULL));

    GdkRGBA transparent = {0.0, 0.0, 0.0, 0.0};
    webkit_web_view_set_background_color(WEBKIT_WEB_VIEW(webview), &transparent);

    const char *html =
        "<!doctype html>"
        "<html>"
        "<head>"
        "<meta charset='utf-8'>"
        "<style>"
        "html, body {"
        "  margin: 0;"
        "  width: 100%;"
        "  height: 100%;"
        "  background: transparent;"
        "  font: 16px system-ui, sans-serif;"
        "}"
        ".panel {"
        "  margin: 48px;"
        "  padding: 24px;"
        "  border-radius: 12px;"
        "  color: white;"
        "  background: rgba(24, 34, 48, 0.58);"
        "  border: 1px solid rgba(255, 255, 255, 0.28);"
        "}"
        ".hole {"
        "  margin-top: 20px;"
        "  height: 120px;"
        "  border: 2px dashed rgba(255,255,255,0.65);"
        "  background: transparent;"
        "}"
        "</style>"
        "</head>"
        "<body>"
        "<div class='panel'>"
        "<h1>GTK4 WebKit Transparency Repro</h1>"
        "<p>The area outside this panel, and the dashed box below, should show the desktop behind the window.</p>"
        "<div class='hole'></div>"
        "</div>"
        "</body>"
        "</html>";

    webkit_web_view_load_html(WEBKIT_WEB_VIEW(webview), html, "file:///");
    gtk_window_set_child(GTK_WINDOW(window), webview);
    gtk_window_present(GTK_WINDOW(window));
    g_printerr("[repro] presented\n");
}

int main(int argc, char **argv) {
    g_printerr("[repro] start\n");
    int gtk_argc = 1;
    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--fullscreen") == 0) {
            fullscreen = TRUE;
            continue;
        }
        argv[gtk_argc++] = argv[i];
    }
    argv[gtk_argc] = NULL;

    GtkApplication *app = gtk_application_new("io.github.willfish.forte.TransparencyRepro", G_APPLICATION_DEFAULT_FLAGS);
    g_signal_connect(app, "activate", G_CALLBACK(on_activate), NULL);
    int status = g_application_run(G_APPLICATION(app), gtk_argc, argv);
    g_printerr("[repro] exit %d\n", status);
    g_object_unref(app);
    return status;
}
