import * as React from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import {
  Boxes,
  BarChart3,
  Building2,
  Eye,
  EyeOff,
  KeyRound,
  Layers,
  LayoutDashboard,
  LogIn,
  LogOut,
  Monitor,
  Moon,
  Sun,
  ScrollText,
  type LucideIcon,
} from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  SidebarProvider,
  Sidebar,
  SidebarHeader,
  SidebarContent,
  SidebarFooter,
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
import { LOCALE_OPTIONS, type Locale, type TranslationKey, useI18n } from "@/lib/i18n";
import { type ThemePreference, useTheme } from "@/lib/theme";
import { cn } from "@/lib/utils";

const NAV: Array<{ exact?: boolean; icon: LucideIcon; labelKey: TranslationKey; to: string }> = [
  { exact: true, icon: LayoutDashboard, labelKey: "nav.overview", to: "/" },
  { icon: Building2, labelKey: "nav.tenants", to: "/tenants" },
  { icon: Boxes, labelKey: "nav.definitions", to: "/resource-definitions" },
  { icon: KeyRound, labelKey: "nav.apiClients", to: "/api-clients" },
  { icon: BarChart3, labelKey: "nav.usage", to: "/usage" },
  { icon: ScrollText, labelKey: "nav.audit", to: "/audit-events" },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const { t } = useI18n();
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
          <SidebarGroup>
            <SidebarGroupLabel>{t("nav.management")}</SidebarGroupLabel>
            <SidebarMenu>
              {NAV.map((item) => {
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
          </SidebarGroup>
        </SidebarContent>

        <SidebarFooter>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                aria-label={t("nav.logout")}
                onClick={() => void logoutAdmin()}
                title={t("nav.logout")}
                tooltip={t("nav.logout")}
              >
                <LogOut />
                <span className="group-data-[state=collapsed]:hidden">{t("nav.logout")}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>
      </Sidebar>

      <SidebarInset>
        <header className="flex h-14 items-center gap-2 border-b px-4">
          <SidebarTrigger />
          <span className="text-sm font-medium">{currentPageLabel}</span>
          <PreferenceControls className="ml-auto" />
        </header>
        <main className="min-w-0 flex-1 overflow-auto p-6">{children}</main>
      </SidebarInset>
    </SidebarProvider>
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
  if (pathname.startsWith("/audit-events")) return t("nav.audit");
  return "";
}
