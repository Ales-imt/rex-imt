import { SignInPage } from '@toolpad/core/SignInPage';
import { useNavigate } from 'react-router';
import {  getCredentialsProvider, type Credentials } from './CredentialProvider';




const localeText = {
  signInTitle: (_orig?: string) => `Connexion`,
  signInSubtitle: "IMT-rex",
  email: "Adresse e-mail",
  password: "Saississez votre mot de passe",
  providerSignInTitle: (_provider: string) => `Se connecter`,
}

export default function SignIn() {
  const navigate = useNavigate();

  return (
    <SignInPage
      providers={[{ id: 'credentials', name: 'Credentials' }]}
      localeText={localeText}
      signIn={async (provider, formData, callbackUrl) => {
        let result: Credentials

        if (provider.id === 'credentials') {
          const email = formData?.get('email') as string;
          const password = formData?.get('password') as string;

          if (!email || !password) {
            return { error: 'Email et password sont requis' };
          }

          result = await getCredentialsProvider().signInWithCredentials(email, password);

          if (result.error) {
            return { error: result.error };
          }

          navigate(callbackUrl || '/');
          return {};

        }
        console.log("erreur car ne doit pas passer par la....")
        return { error: 'credential non pris en charge' }
      }
      }
    />
  );
}



