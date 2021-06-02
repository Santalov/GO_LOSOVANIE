import React from 'react';
import {createStyles, makeStyles, TextField, Theme} from '@material-ui/core';
import AppButtonContainer from '../../components/app-button-container/app-button-container';
import AppButton from '../../components/app-button/app-button';
import AppHeadline from '../../components/app-headline/app-headline';
import * as yup from 'yup';
import {useFormik} from 'formik';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    field: {
      marginBottom: theme.spacing(2)
    },
    lastField: {
      marginBottom: theme.spacing(3)
    }
  })
);

const addrLen = 33;

const validationSchema = yup.object({
  receiver: yup
    .string()
    .min(addrLen, `Длина адреса ровно ${addrLen} символа`)
    .max(addrLen, `Длина адреса ровно ${addrLen} символа`)
    .required('Введите адрес получателя'),
  amount: yup
    .number()
    .required('Введите сумму'),
});

function AppSendPage() {
  const classes = useStyles();
  const formik = useFormik({
    initialValues: {
      receiver: '',
      amount: '',
    },
    validationSchema: validationSchema,
    onSubmit: (values) => {
      alert(JSON.stringify(values));
    }
  });
  return (
    <>
      <AppHeadline>
        Отправить
      </AppHeadline>
      <form onSubmit={formik.handleSubmit}>
        <TextField
          required
          id="receiver"
          label="Адрес получателя"
          placeholder="hex строка длиной 33 символа"
          type="text"
          fullWidth
          variant="outlined"
          className={classes.field}
          value={formik.values.receiver}
          onChange={formik.handleChange}
          error={formik.touched.receiver && !!formik.errors.receiver}
          helperText={formik.touched.receiver && formik.errors.receiver}
        >
        </TextField>
        <TextField
          required
          id="amount"
          label="Сумма"
          placeholder="Целое число"
          type="number"
          fullWidth
          variant="outlined"
          className={classes.lastField}
          value={formik.values.amount}
          onChange={formik.handleChange}
          error={formik.touched.amount && !!formik.errors.amount}
          helperText={formik.touched.amount && formik.errors.amount}
        >
        </TextField>
        <AppButtonContainer>
          <AppButton
            type="submit"
          >
            Отправить
          </AppButton>
        </AppButtonContainer>
      </form>
    </>
  )
}

export default AppSendPage
