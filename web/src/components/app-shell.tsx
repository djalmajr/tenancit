import * as React from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import {
  Eye,
  EyeOff,
  KeyRound,
  Layers,
  LogIn,
  LogOut,
  Monitor,
  Moon,
  Sun,
} from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  SidebarProvider,
  Sidebar,
  SidebarHeader,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarInset,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import {
  ADMIN_TOKEN_CHANGE_EVENT,
  api,
  clearAdminToken,
  consumePendingAdminAuthMessage,
  fetchAdminAuthConfig,
  fetchAdminSession,
  getAdminToken,
  logoutAdminSession,
  setAdminToken,
  type AdminAuthConfig,
  type AdminAuthMessage,
  type AdminSession,
} from "@/lib/api";
import { avatarHue, initials } from "@/lib/avatar";
import { LOCALE_OPTIONS, LOCALE_STORAGE_KEY, type Locale, type TranslationKey, useI18n } from "@/lib/i18n";
import { type ThemePreference, useTheme } from "@/lib/theme";
import { cn } from "@/lib/utils";
import { AdminCapabilitiesProvider } from "@/lib/admin-capabilities";
import { ALL_ADMIN_PERMISSIONS } from "@/lib/admin-permissions";
import { visibleNavGroups } from "@/lib/admin-navigation";

export function AppShell({ children }: { children: React.ReactNode }) {
  const { setLocale, t } = useI18n();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const [authMessageKey, setAuthMessageKey] = React.useState<TranslationKey | "">("");
  const [draftAdminToken, setDraftAdminToken] = React.useState(() => getAdminToken());
  const [hasAdminToken, setHasAdminToken] = React.useState(() => Boolean(getAdminToken()));
  const [authConfig, setAuthConfig] = React.useState<AdminAuthConfig>();
  const [adminSession, setAdminSessionState] = React.useState<AdminSession>();
  const [authLoadError, setAuthLoadError] = React.useState(false);

  React.useEffect(() => {
    const controller = new AbortController();
    async function loadAuthentication() {
      try {
        const config = await fetchAdminAuthConfig(controller.signal);
        setAuthConfig(config);
        if (config.mode === "oidc") {
          clearAdminToken();
          const session = await fetchAdminSession(controller.signal);
          setAdminSessionState(session);
        }
      } catch (error) {
        if (!(error instanceof DOMException && error.name === "AbortError")) {
          setAuthLoadError(true);
        }
      }
    }
    void loadAuthentication();
    return () => controller.abort();
  }, []);

  React.useEffect(() => {
    const authenticated = authConfig?.mode === "oidc" ? Boolean(adminSession) : hasAdminToken;
    if (!authenticated || localStorage.getItem(LOCALE_STORAGE_KEY)) return;
    const controller = new AbortController();
    void api.getSettings(controller.signal).then((snapshot) => {
      const configured = snapshot.values.default_locale;
      if (LOCALE_OPTIONS.some((option) => option.value === configured)) setLocale(configured as Locale);
    }).catch(() => undefined);
    return () => controller.abort();
  }, [adminSession, authConfig?.mode, hasAdminToken, setLocale]);

  React.useEffect(() => {
    function showAdminAuth(messageKey: AdminAuthMessage) {
      setAuthMessageKey(messageKey);
      if (authConfig?.mode === "oidc") {
        setAdminSessionState(undefined);
        return;
      }
      setDraftAdminToken("");
      clearAdminToken();
      setHasAdminToken(false);
    }

    function requestAdminAuth(event: Event) {
      const detail: unknown = event instanceof CustomEvent ? event.detail : undefined;
      const messageKey =
        typeof detail === "object" && detail !== null && "messageKey" in detail
          ? detail.messageKey
          : undefined;
      const pendingMessage = consumePendingAdminAuthMessage();
      showAdminAuth(
        messageKey === "auth.invalidToken" ||
          messageKey === "auth.requiredAccess" ||
          messageKey === "auth.sessionExpired"
          ? messageKey
          : pendingMessage ?? "auth.requiredAccess",
      );
    }
    window.addEventListener("admin-auth-required", requestAdminAuth);
    const pendingMessage = consumePendingAdminAuthMessage();
    if (pendingMessage) showAdminAuth(pendingMessage);
    return () => window.removeEventListener("admin-auth-required", requestAdminAuth);
  }, [authConfig?.mode]);

  React.useEffect(() => {
    let remountTimer: number | undefined;
    function syncAdminTokenState() {
      const hasNextCredential = Boolean(getAdminToken());
      window.clearTimeout(remountTimer);
      // Force a committed unauthenticated boundary before observers can mount
      // for a replacement identity. The query-cache listener cancels/clears
      // the old identity synchronously in the same event.
      setHasAdminToken(false);
      if (hasNextCredential) {
        remountTimer = window.setTimeout(() => setHasAdminToken(true), 0);
      }
    }
    window.addEventListener(ADMIN_TOKEN_CHANGE_EVENT, syncAdminTokenState);
    window.addEventListener("storage", syncAdminTokenState);
    return () => {
      window.clearTimeout(remountTimer);
      window.removeEventListener(ADMIN_TOKEN_CHANGE_EVENT, syncAdminTokenState);
      window.removeEventListener("storage", syncAdminTokenState);
    };
  }, []);

  const currentPageLabel = pageLabel(pathname, t);
  const currentPageDescription = pageDescription(pathname, t);
  const permissions = authConfig?.mode === "oidc" ? adminSession?.permissions ?? [] : ALL_ADMIN_PERMISSIONS;

  function saveAdminToken(event?: React.FormEvent<HTMLFormElement>) {
    event?.preventDefault();
    const token = draftAdminToken.trim();
    if (!token) {
      clearAdminToken();
      return;
    }
    setAdminToken(token);
    setAuthMessageKey("");
    window.location.reload();
  }

  async function logoutAdmin() {
    if (authConfig?.mode === "oidc") {
      await logoutAdminSession();
      setAdminSessionState(undefined);
      return;
    }
    clearAdminToken();
    setDraftAdminToken("");
    window.location.reload();
  }

  if (authLoadError) {
    return <AdminAuthUnavailableScreen />;
  }

  if (!authConfig) {
    return <AdminAuthLoadingScreen />;
  }

  if (authConfig.mode === "oidc" && !adminSession) {
    return (
      <OIDCAccessScreen
        authMessage={authMessageKey ? t(authMessageKey) : ""}
        loginURL={authConfig.login_url ?? "/v1/auth/login"}
        returnTo={pathname}
      />
    );
  }

  if (authConfig.mode === "legacy_shared_token" && !hasAdminToken) {
    return (
      <AdminAccessScreen
        authMessage={authMessageKey ? t(authMessageKey) : ""}
        draftAdminToken={draftAdminToken}
        onDraftAdminTokenChange={setDraftAdminToken}
        onSubmit={saveAdminToken}
      />
    );
  }

  return (
    <AdminCapabilitiesProvider permissions={permissions}>
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader>
          <div className="flex items-center gap-2 px-1 py-1.5">
            <div className="flex size-8 shrink-0 items-center justify-center">
              <Layers className="size-[22px]" />
            </div>
            <div className="flex flex-col group-data-[state=collapsed]:hidden">
              <span className="text-sm font-semibold leading-tight">{t("app.title")}</span>
              <span className="text-xs leading-tight text-sidebar-foreground/60">{t("app.subtitle")}</span>
            </div>
          </div>
        </SidebarHeader>

        <SidebarContent>
          {visibleNavGroups(new Set(permissions)).map((group) => <SidebarGroup key={group.labelKey}>
            <SidebarGroupLabel>{t(group.labelKey)}</SidebarGroupLabel>
            <SidebarMenu>
              {group.items.map((item) => {
                const active = item.exact ? pathname === item.to : pathname.startsWith(item.to);
                const Icon = item.icon;
                const label = t(item.labelKey);
                return (
                  <SidebarMenuItem key={item.to}>
                    <SidebarMenuButton
                      isActive={active}
                      render={<Link aria-label={label} to={item.to} />}
                      tooltip={label}
                    >
                      <Icon />
                      <span className="group-data-[state=collapsed]:hidden">{label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </SidebarGroup>)}
        </SidebarContent>

      </Sidebar>

      <SidebarInset>
        <header className="flex h-14 items-center gap-2 border-b px-4">
          <SidebarTrigger />
          <div className="flex min-w-0 flex-col">
            <span className="truncate text-sm font-medium leading-tight">{currentPageLabel}</span>
            {currentPageDescription && (
              <span className="truncate text-xs leading-tight text-muted-foreground">
                {currentPageDescription}
              </span>
            )}
          </div>
          <div className="ml-auto flex items-center gap-1.5">
            <PreferenceControls />
            <UserMenu
              session={adminSession}
              sharedCredential={authConfig.mode === "legacy_shared_token"}
              onLogout={() => void logoutAdmin()}
            />
          </div>
        </header>
        <main className="min-w-0 flex-1 overflow-auto p-6">{children}</main>
      </SidebarInset>
    </SidebarProvider>
    </AdminCapabilitiesProvider>
  );
}

function UserMenu({
  onLogout,
  session,
  sharedCredential,
}: {
  onLogout: () => void;
  session?: AdminSession;
  sharedCredential: boolean;
}) {
  const { t } = useI18n();
  const displayName = session?.label || t("auth.sharedCredential");
  const detail = sharedCredential
    ? t("auth.sharedToken")
    : session?.roles.length
      ? session.roles.join(", ")
      : t("auth.oidcSession");
  const avatarLabel = initials(displayName);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={t("nav.account")}
        className={cn(
          "inline-flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-full outline-none",
          "ring-offset-background transition-opacity hover:opacity-90",
          "focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        )}
        title={displayName}
        type="button"
      >
        <span
          aria-hidden
          className="flex size-8 items-center justify-center rounded-full text-[11px] font-semibold leading-none text-white"
          style={{ background: avatarHue(session?.subject || displayName) }}
        >
          {avatarLabel}
        </span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="min-w-56">
        <DropdownMenuGroup>
          <DropdownMenuLabel className="font-normal">
            <div className="flex items-center gap-3 py-0.5">
              <span
                aria-hidden
                className="flex size-9 shrink-0 items-center justify-center rounded-full text-xs font-semibold leading-none text-white"
                style={{ background: avatarHue(session?.subject || displayName) }}
              >
                {avatarLabel}
              </span>
              <div className="min-w-0">
                <div className="truncate text-sm font-medium leading-tight">{displayName}</div>
                <div className="truncate text-xs text-muted-foreground">{detail}</div>
              </div>
            </div>
          </DropdownMenuLabel>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem onClick={onLogout}>
            <LogOut />
            {t("nav.logout")}
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function AdminAuthLoadingScreen() {
  const { t } = useI18n();
  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-sidebar p-4">
      <div aria-live="polite" className="flex items-center gap-3 text-sm text-muted-foreground" role="status">
        <span className="size-4 animate-spin rounded-full border-2 border-current border-r-transparent" />
        {t("auth.loading")}
      </div>
    </div>
  );
}

function AdminAuthUnavailableScreen() {
  const { t } = useI18n();
  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-sidebar p-4">
      <div className="w-full max-w-md rounded-md border border-border bg-background p-5 shadow-sm">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-destructive/10 text-destructive">
              <KeyRound className="size-5" />
            </div>
            <div className="min-w-0">
              <h1 className="text-base font-semibold">{t("auth.title")}</h1>
              <p className="text-sm text-muted-foreground">{t("auth.unavailable")}</p>
            </div>
          </div>
          <PreferenceControls compact />
        </div>
        <Button className="mt-5 w-full" onClick={() => window.location.reload()} type="button">
          {t("auth.retry")}
        </Button>
      </div>
    </div>
  );
}

function OIDCAccessScreen({
  authMessage,
  loginURL,
  returnTo,
}: {
  authMessage: string;
  loginURL: string;
  returnTo: string;
}) {
  const { t } = useI18n();

  function startLogin() {
    const target = new URL(loginURL, window.location.origin);
    target.searchParams.set("return_to", returnTo || "/");
    window.location.assign(target.toString());
  }

  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-sidebar p-4">
      <div className="w-full max-w-md rounded-md border border-border bg-background p-5 shadow-sm">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
              <LogIn className="size-5" />
            </div>
            <div className="min-w-0">
              <h1 className="text-base font-semibold">{t("auth.title")}</h1>
              <p className="text-sm text-muted-foreground">
                {authMessage || t("auth.oidcDescription")}
              </p>
            </div>
          </div>
          <PreferenceControls compact />
        </div>
        <Button className="mt-5 w-full" onClick={startLogin} type="button">
          <LogIn className="size-4" />
          {t("auth.oidcEnter")}
        </Button>
      </div>
    </div>
  );
}

function AdminAccessScreen({
  authMessage,
  draftAdminToken,
  onDraftAdminTokenChange,
  onSubmit,
}: {
  authMessage: string;
  draftAdminToken: string;
  onDraftAdminTokenChange: (token: string) => void;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
}) {
  const { t } = useI18n();
  const [showToken, setShowToken] = React.useState(false);
  const tokenInputRef = React.useRef<HTMLInputElement>(null);
  React.useEffect(() => tokenInputRef.current?.focus(), []);
  const disabled = !draftAdminToken.trim();
  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-sidebar p-4">
      <form className="w-full max-w-md rounded-md border border-border bg-background p-5 shadow-sm" onSubmit={onSubmit}>
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex size-10 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
              <KeyRound className="size-5" />
            </div>
            <div className="min-w-0">
              <h1 className="text-base font-semibold">{t("auth.title")}</h1>
              <p className="text-sm text-muted-foreground">
                {authMessage || t("auth.description")}
              </p>
            </div>
          </div>
          <PreferenceControls compact />
        </div>

        <label className="mt-5 block text-sm font-medium" htmlFor="admin-token">
          {t("auth.token")}
        </label>
        <div className="relative mt-2">
          <KeyRound className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            autoComplete="off"
            className="pl-7 pr-7"
            id="admin-token"
            ref={tokenInputRef}
            onChange={(event) => onDraftAdminTokenChange(event.target.value)}
            type={showToken ? "text" : "password"}
            value={draftAdminToken}
          />
          <button
            aria-label={showToken ? t("auth.hideToken") : t("auth.showToken")}
            className="absolute right-1.5 top-1/2 inline-flex size-6 -translate-y-1/2 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground"
            onClick={() => setShowToken((value) => !value)}
            type="button"
          >
            {showToken ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
          </button>
        </div>

        <Button className="mt-4 w-full" disabled={disabled} type="submit">
          <LogIn className="size-4" />
          {t("auth.enter")}
        </Button>
      </form>
    </div>
  );
}

function PreferenceControls({ className, compact = false }: { className?: string; compact?: boolean }) {
  const { locale, setLocale } = useI18n();

  return (
    <div className={cn("flex shrink-0 items-center", compact ? "gap-1.5" : "gap-2", className)}>
      <LanguageMenu locale={locale} setLocale={setLocale} />
      <ThemeMenu />
    </div>
  );
}

function LanguageMenu({
  locale,
  setLocale,
}: {
  locale: Locale;
  setLocale: (locale: Locale) => void;
}) {
  const { t } = useI18n();
  const selected = LOCALE_OPTIONS.find((option) => option.value === locale) ?? LOCALE_OPTIONS[0];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={t("preferences.language")}
        className={cn(buttonVariants({ size: "icon-sm", variant: "ghost" }))}
        title={t("preferences.language")}
        type="button"
      >
        <span aria-hidden="true" className="text-xl leading-none">{selected.flag}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="min-w-max">
        <DropdownMenuRadioGroup onValueChange={(value) => setLocale(value as Locale)} value={locale}>
          {LOCALE_OPTIONS.map((option) => (
            <DropdownMenuRadioItem className="gap-2 whitespace-nowrap pr-8" key={option.value} value={option.value}>
              <span aria-hidden="true" className="text-lg leading-none">{option.flag}</span>
              <span className="pr-1">{option.label}</span>
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function ThemeMenu() {
  const { t } = useI18n();
  const { resolvedTheme, setTheme, theme } = useTheme();
  const options: Array<{ label: string; value: ThemePreference }> = [
    { label: t("preferences.theme.light"), value: "light" },
    { label: t("preferences.theme.dark"), value: "dark" },
    { label: t("preferences.theme.system"), value: "system" },
  ];
  const currentLabel = options.find((option) => option.value === theme)?.label ?? t("preferences.theme.system");
  const ThemeIcon = theme === "system" ? Monitor : resolvedTheme === "dark" ? Moon : Sun;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={`${t("preferences.theme")}: ${currentLabel}`}
        className={cn(buttonVariants({ size: "icon-sm", variant: "ghost" }))}
        title={`${t("preferences.theme")}: ${currentLabel}`}
        type="button"
      >
        <ThemeIcon className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent className="min-w-24">
        <DropdownMenuRadioGroup onValueChange={(value) => setTheme(value as ThemePreference)} value={theme}>
          {options.map((option) => (
            <DropdownMenuRadioItem key={option.value} value={option.value}>
              {option.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function pageLabel(pathname: string, t: (key: TranslationKey) => string): string {
  if (pathname === "/") return t("nav.overview");
  if (pathname.startsWith("/tenants")) return t("nav.tenants");
  if (pathname.startsWith("/resource-definitions")) return t("nav.definitions");
  if (pathname.startsWith("/api-clients")) return t("nav.apiClients");
  if (pathname.startsWith("/usage")) return t("nav.usage");
  if (pathname.startsWith("/operations/health")) return t("nav.health");
  if (pathname.startsWith("/audit-events")) return t("nav.audit");
  if (pathname.startsWith("/security/sessions")) return t("nav.sessions");
  if (pathname.startsWith("/operations/settings")) return t("nav.settings");
  if (pathname.startsWith("/integrations/webhooks")) return t("nav.integrations");
  return "";
}

function pageDescription(pathname: string, t: (key: TranslationKey) => string): string {
  if (pathname === "/") return t("overview.description");
  if (pathname === "/tenants") return t("tenants.description");
  if (pathname === "/resource-definitions") return t("definitions.description");
  if (pathname === "/api-clients") return t("apiClients.description");
  if (pathname === "/usage") return t("usage.description");
  if (pathname.startsWith("/operations/health")) return t("operationsHealth.description");
  if (pathname.startsWith("/audit-events")) return t("audit.description");
  if (pathname.startsWith("/security/sessions")) return t("sessions.description");
  if (pathname.startsWith("/integrations/webhooks")) return t("integrations.description");
  if (pathname.startsWith("/operations/settings")) return t("settings.description");
  return "";
}
