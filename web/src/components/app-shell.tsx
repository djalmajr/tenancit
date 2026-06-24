import * as React from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import {
  Boxes,
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
import { clearAdminToken, getAdminToken, setAdminToken } from "@/lib/api";
import { LOCALE_OPTIONS, type Locale, type TranslationKey, useI18n } from "@/lib/i18n";
import { type ThemePreference, useTheme } from "@/lib/theme";
import { cn } from "@/lib/utils";

const NAV: Array<{ exact?: boolean; icon: LucideIcon; labelKey: TranslationKey; to: string }> = [
  { exact: true, icon: LayoutDashboard, labelKey: "nav.overview", to: "/" },
  { icon: Building2, labelKey: "nav.tenants", to: "/tenants" },
  { icon: Boxes, labelKey: "nav.definitions", to: "/resource-definitions" },
  { icon: KeyRound, labelKey: "nav.apiClients", to: "/api-clients" },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const { t } = useI18n();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const [authMessageKey, setAuthMessageKey] = React.useState<TranslationKey | "">("");
  const [draftAdminToken, setDraftAdminToken] = React.useState(() => getAdminToken());
  const [hasAdminToken, setHasAdminToken] = React.useState(() => Boolean(getAdminToken()));

  React.useEffect(() => {
    function requestAdminAuth() {
      setAuthMessageKey("auth.requiredAccess");
      setDraftAdminToken("");
      clearAdminToken();
      setHasAdminToken(false);
    }
    window.addEventListener("admin-auth-required", requestAdminAuth);
    return () => window.removeEventListener("admin-auth-required", requestAdminAuth);
  }, []);

  React.useEffect(() => {
    function syncAdminTokenState() {
      setHasAdminToken(Boolean(getAdminToken()));
    }
    window.addEventListener("admin-token-change", syncAdminTokenState);
    return () => window.removeEventListener("admin-token-change", syncAdminTokenState);
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

  function logoutAdmin() {
    clearAdminToken();
    setDraftAdminToken("");
    window.location.reload();
  }

  if (!hasAdminToken) {
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
                    <SidebarMenuButton asChild isActive={active} tooltip={label}>
                      <Link aria-label={label} to={item.to}>
                        <Icon />
                        <span className="group-data-[state=collapsed]:hidden">{label}</span>
                      </Link>
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
              <SidebarMenuButton aria-label={t("nav.logout")} onClick={logoutAdmin} title={t("nav.logout")} tooltip={t("nav.logout")}>
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
        <main className="flex-1 overflow-auto p-6">{children}</main>
      </SidebarInset>
    </SidebarProvider>
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
          <KeyRound className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            autoComplete="off"
            autoFocus
            className="h-10 pl-9 pr-10"
            id="admin-token"
            onChange={(event) => onDraftAdminTokenChange(event.target.value)}
            type={showToken ? "text" : "password"}
            value={draftAdminToken}
          />
          <button
            aria-label={showToken ? t("auth.hideToken") : t("auth.showToken")}
            className="absolute right-2 top-1/2 inline-flex size-7 -translate-y-1/2 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground"
            onClick={() => setShowToken((value) => !value)}
            type="button"
          >
            {showToken ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
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
            <DropdownMenuRadioItem className="gap-2 whitespace-nowrap pr-3" key={option.value} value={option.value}>
              <span aria-hidden="true" className="text-lg leading-none">{option.flag}</span>
              <span>{option.label}</span>
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
  return "";
}
