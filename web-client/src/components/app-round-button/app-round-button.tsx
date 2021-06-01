import React from "react";
import makeStyles from "@material-ui/core/styles/makeStyles";
import withTheme from "@material-ui/core/styles/withTheme";
import ButtonBase from "@material-ui/core/ButtonBase";

function AppRoundButtonRaw(props) {
  const classes = makeStyles({
    root: {
      position: "relative",
      border: "1px solid",
      borderColor: props.theme.palette.primary.main,
      borderRadius: "50%",
      flexShrink: 0,
      height: 36,
      width: 36,
      lineHeight: "34px",
      paddingLeft: 0,
      paddingRight: 0,
      paddingTop: 0,
      paddingBottom: 0,
      fontSize: "1rem",
      color: props.theme.palette.primary.main,
    },
    disabled: {
      borderColor: props.theme.palette.text.disabled,
      color: props.theme.palette.text.secondary,
    },
  })();
  return (
    <ButtonBase
      {...props}
      className={
        classes.root +
        (props.disabled ? " " + classes.disabled : "") +
        (props.className ? " " + props.className : "")
      }
    >
      {props.children}
    </ButtonBase>
  );
}

const AppRoundButton = withTheme(AppRoundButtonRaw);

export default AppRoundButton;
