import React from "react";
import {ThemeProvider} from "@material-ui/core/styles";
import appLightTheme from "./app-light-theme";
import appDarkTheme from "./app-dark-theme";
import themeService from '../services/theme.service';

class AppThemeProvider extends React.Component {
  state: { isDark: boolean };

  constructor(props) {
    super(props);
    this.state = {
      isDark: themeService.isDarkTheme,
    };
    themeService.toggleTheme = this.toggleTheme;
  }

  toggleTheme = () => {
    themeService.isDarkTheme = !this.state.isDark;
    this.setState({
      isDark: !this.state.isDark,
    });
  };

  render():
    | React.ReactElement<any, string | React.JSXElementConstructor<any>>
    | string
    | number
    | {}
    | React.ReactNodeArray
    | React.ReactPortal
    | boolean
    | null
    | undefined {
    return (
      <ThemeProvider theme={this.state.isDark ? appDarkTheme : appLightTheme}>
        {this.props.children}
      </ThemeProvider>
    );
  }
}

export default AppThemeProvider;
