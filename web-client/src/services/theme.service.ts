class ThemeService {
  // initialized inside AppThemeProvider
  public toggleTheme: Function = () => {
    console.error("toggle theme is called before init");
  };
  // this flag changes inside toggleTheme callback, which is
  // inside AppThemeProvider
  public isDarkTheme = true;
}

const themeService = new ThemeService();

export default themeService;
