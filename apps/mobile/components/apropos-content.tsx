import { useTheme } from '@/hooks/use-theme';
import { Linking, ScrollView, StyleSheet, Text, TouchableOpacity, View } from 'react-native';

// ─── Types ────────────────────────────────────────────────────────────────────

type SectionProps = {
  title: string;
  children: React.ReactNode;
  colors: ReturnType<typeof useTheme>;
};

type RowProps = {
  label: string;
  value: string;
  colors: ReturnType<typeof useTheme>;
};

// ─── Composants internes ──────────────────────────────────────────────────────

function Section({ title, children, colors }: SectionProps) {
  return (
    <View style={styles.section}>
      <Text style={[styles.sectionTitle, { color: colors.tint }]}>{title}</Text>
      {children}
    </View>
  );
}

function Row({ label, value, colors }: RowProps) {
  return (
    <View style={[styles.row, { borderBottomColor: colors.dividerLine }]}>
      <Text style={[styles.rowLabel, { color: colors.textSecondary }]}>{label}</Text>
      <Text style={[styles.rowValue, { color: colors.textPrimary }]}>{value}</Text>
    </View>
  );
}

function InfoBox({ children, colors }: { children: React.ReactNode; colors: ReturnType<typeof useTheme> }) {
  return (
    <View style={[styles.infoBox, { backgroundColor: colors.tint + '15', borderLeftColor: colors.tint }]}>
      {children}
    </View>
  );
}

// ─── Contenu partagé ─────────────────────────────────────────────────────────

type AproposContentProps = {
  colors: ReturnType<typeof useTheme>;
  children?: React.ReactNode;
};

export function AproposContent({ colors, children }: AproposContentProps) {
  return (
    <ScrollView contentContainerStyle={styles.scroll} showsVerticalScrollIndicator={false}>

      {/* En-tête */}
      <View style={styles.header}>
        <View style={[styles.appIcon, { backgroundColor: colors.tint }]}>
          <Text style={styles.appIconText}>A</Text>
        </View>
        <Text style={[styles.appName, { color: colors.textPrimary }]}>rex-imt</Text>
        <Text style={[styles.appTagline, { color: colors.textSecondary }]}>
          Retours d'expérience — IMT Mines Alès
        </Text>
      </View>

      {/* ── 1. Responsable de traitement ───────────────────────────────── */}
      <Section title="Responsable de traitement" colors={colors}>
        <Row label="Établissement" value="IMT Mines Alès" colors={colors} />
        <Row label="Adresse" value="6 avenue de Clavières, 30319 Alès Cedex" colors={colors} />
        <Row label="Contact DPO" value="dpo@mines-ales.fr" colors={colors} />
        <TouchableOpacity onPress={() => Linking.openURL('mailto:dpo@mines-ales.fr')}>
          <Text style={[styles.link, { color: colors.tint }]}>
            Contacter le délégué à la protection des données →
          </Text>
        </TouchableOpacity>
      </Section>

      {/* ── 2. Données collectées et finalités ─────────────────────────── */}
      <Section title="Données collectées et finalités" colors={colors}>
        <Text style={[styles.bodyText, { color: colors.textPrimary }]}>
          Cette application collecte les données suivantes dans le cadre de sa mission pédagogique :
        </Text>

        <View style={styles.dataList}>
          {[
            { cat: 'Identité', detail: 'Nom, prénom, adresse e-mail institutionnelle — utilisés pour l\'authentification via le LDAP de l\'école.' },
            { cat: 'Feedbacks', detail: 'Contenu textuel, pseudo choisi, promotion et groupe — collectés pour améliorer la qualité des enseignements. Chaque message est relu par un modérateur avant toute diffusion ou analyse (voir ci-dessous).' },
            { cat: 'Évaluations', detail: 'Scores par dimension pédagogique et verbatims optionnels. Comme les feedbacks, chaque verbatim est relu par un modérateur avant toute diffusion ou analyse, et l\'évaluation est associée à une donnée d\'identification chiffrée (adresse IP et identifiant technique) conservée pour répondre aux obligations légales ; elle n\'est ni affichée ni accessible à l\'équipe pédagogique.' },
            { cat: 'Présences', detail: 'Identifiant de séance, statut (présent / retard) et heure de pointage — collectés par scan de QR code lors des séances. Un registre cryptographique d\'intégrité garantit l\'inaltérabilité de ces données. Base légale : obligation légale d\'assiduité (Art. L123-1 Code de l\'éducation).' },
            { cat: 'Données techniques', detail: 'Adresse IP et identifiant technique chiffrés — conservés pour répondre aux obligations légales (voir ci-dessous).' },
          ].map(item => (
            <View key={item.cat} style={[styles.dataItem, { borderLeftColor: colors.tint + '66' }]}>
              <Text style={[styles.dataCat, { color: colors.tint }]}>{item.cat}</Text>
              <Text style={[styles.dataDetail, { color: colors.textPrimary }]}>{item.detail}</Text>
            </View>
          ))}
        </View>

        <Text style={[styles.legalBase, { color: colors.textSecondary }]}>
          Base légale principale : mission de service public d'enseignement supérieur (Art. 6.1.e RGPD). Pour les données de présence : obligation légale d'assiduité (Art. 6.1.c RGPD — Art. L123-1 Code de l'éducation).
        </Text>
      </Section>

      {/* ── 3. Durées de conservation ───────────────────────────────────── */}
      <Section title="Durées de conservation" colors={colors}>
        <Row label="Accès à votre compte" value="Coupé 1 an après votre départ" colors={colors} />
        <Row label="Votre identité" value="Anonymisée 10 ans après votre départ" colors={colors} />
        <Row label="Contenu de vos feedbacks" value="3 ans à compter de la publication" colors={colors} />
        <Row label="Messages refusés" value="Chiffrés, puis supprimés après 90 jours" colors={colors} />
        <Row label="Données d'identification (IP chiffrée)" value="1 an à compter de la publication" colors={colors} />
        <Row label="Évaluations" value="Conservées à des fins statistiques" colors={colors} />
        <Row label="Données de présence" value="Conservées ; identité anonymisée après 10 ans" colors={colors} />
        <Row label="Tokens de session" value="3 mois puis suppression automatique" colors={colors} />
      </Section>

      {/* ── 3b. Modération avant diffusion ─────────────────────────────── */}
      <Section title="Modération de vos messages" colors={colors}>
        <InfoBox colors={colors}>
          <Text style={[styles.infoTitle, { color: colors.tint }]}>
            ℹ️  Une relecture humaine avant publication
          </Text>
          <Text style={[styles.infoText, { color: colors.textPrimary }]}>
            Chaque feedback que vous envoyez est d'abord relu par un modérateur. Tant qu'il n'est pas publié, votre message n'est ni diffusé à l'équipe pédagogique ni analysé par le système de classification automatique.
          </Text>
          <Text style={[styles.infoText, { color: colors.textPrimary, marginTop: 8 }]}>
            Le texte d'origine sert uniquement à cette relecture : il n'est jamais diffusé et il est effacé au moment de la publication. Le modérateur n'a accès à aucune donnée permettant de vous identifier. Vous pouvez suivre le statut de vos messages (en attente, publié ou refusé avec son motif) directement dans l'application.
          </Text>
          <Text style={[styles.infoText, { color: colors.textPrimary, marginTop: 8 }]}>
            Si votre message est refusé, son texte d'origine est immédiatement chiffré (illisible au repos, y compris par nos équipes) et conservé au maximum 90 jours — le temps d'une éventuelle contestation — avant d'être supprimé définitivement. Un message refusé n'ayant jamais été publié, il n'est soumis à aucune obligation de conservation.
          </Text>
          <Text style={[styles.infoText, { color: colors.textPrimary, marginTop: 8 }]}>
            Les verbatims que vous laissez dans une évaluation de cours suivent exactement le même parcours.
          </Text>
        </InfoBox>
      </Section>

      {/* ── 4. Obligation LCEN ─────────────────────────────────────────── */}
      <Section title="Conservation légale des contenus (LCEN)" colors={colors}>
        <InfoBox colors={colors}>
          <Text style={[styles.infoTitle, { color: colors.tint }]}>
            ℹ️  Ce que cela signifie pour vos messages
          </Text>
          <Text style={[styles.infoText, { color: colors.textPrimary }]}>
            Conformément à l'article 6-II de la loi n° 2004-575 (LCEN) et au décret n° 2021-1362 du 20 octobre 2021, les données techniques permettant d'identifier l'auteur d'un contenu publié en ligne (adresse IP chiffrée, identifiant technique) sont conservées pendant <Text style={{ fontWeight: '700' }}>1 an</Text> à compter de la publication.
          </Text>
          <Text style={[styles.infoText, { color: colors.textPrimary, marginTop: 8 }]}>
            Si vous demandez la suppression d'un feedback, son contenu sera immédiatement effacé de l'affichage. Les données techniques resteront conservées jusqu'à l'expiration du délai légal d'un an, puis seront automatiquement détruites.
          </Text>
        </InfoBox>
      </Section>

      {/* ── 4b. Registre d'intégrité des présences ─────────────────────── */}
      <Section title="Registre de présences et intégrité" colors={colors}>
        <InfoBox colors={colors}>
          <Text style={[styles.infoTitle, { color: colors.tint }]}>
            ℹ️  Pourquoi vos présences ne peuvent pas être supprimées
          </Text>
          <Text style={[styles.infoText, { color: colors.textPrimary }]}>
            Chaque pointage est enregistré dans un registre cryptographique dont les entrées sont chaînées par empreinte SHA-256. Toute modification ou suppression rompt la chaîne et est immédiatement détectable. Ce mécanisme garantit l'authenticité des relevés d'assiduité conformément à l'obligation légale (Art. L123-1 Code de l'éducation).
          </Text>
          <Text style={[styles.infoText, { color: colors.textPrimary, marginTop: 8 }]}>
            En application de l'Art. 17.3.b du RGPD, le droit à l'effacement ne s'applique pas à ces données pendant la durée de l'obligation légale. À l'issue de cette période, vos données nominatives sont anonymisées dans notre base ; les empreintes cryptographiques du registre sont conservées sans lien avec votre identité.
          </Text>
        </InfoBox>
      </Section>

      {/* ── 4c. Cycle de vie du compte ─────────────────────────────────── */}
      <Section title="Ce que devient votre compte après votre départ" colors={colors}>
        <InfoBox colors={colors}>
          <Text style={[styles.infoTitle, { color: colors.tint }]}>
            ℹ️  Deux étapes
          </Text>
          <Text style={[styles.infoText, { color: colors.textPrimary }]}>
            {"Un an après votre départ de l'école, l'accès à votre compte est coupé et vos sessions sont fermées. Vous ne pouvez plus vous connecter."}
          </Text>
          <Text style={[styles.infoText, { color: colors.textPrimary, marginTop: 8 }]}>
            {"Votre nom, votre prénom et votre adresse e-mail sont en revanche conservés au-delà : ce sont eux qui permettent de rattacher vos relevés de présence à une personne. L'école doit pouvoir justifier la réalisation de votre formation auprès de ses financeurs — organismes de financement, administration fiscale, ou fonds européens selon les cas."}
          </Text>
          <Text style={[styles.infoText, { color: colors.textPrimary, marginTop: 8 }]}>
            <Text style={{ fontWeight: '700' }}>Dix ans</Text>
            {" après votre départ, cette identité est effacée : nom et prénom vidés, adresse e-mail remplacée. Vos relevés de présence subsistent alors sans aucun lien avec vous. Ce délai correspond à la plus longue des obligations applicables ; il est appliqué à tous par prudence, faute de pouvoir distinguer au cas par cas le mode de financement de chaque formation."}
          </Text>
        </InfoBox>
      </Section>

      {/* ── 5. Destinataires ───────────────────────────────────────────── */}
      <Section title="Destinataires des données" colors={colors}>
        <Text style={[styles.bodyText, { color: colors.textPrimary }]}>
          Vos données sont accessibles uniquement aux personnes habilitées suivantes :
        </Text>
        {[
          'Équipe pédagogique et administrative de l\'IMT Mines Alès (feedbacks modérés)',
          'Administrateurs de la plateforme (gestion des comptes)',
          'Système de classification automatique par intelligence artificielle hébergé sur l\'infrastructure de l\'école (analyse des feedbacks)',
        ].map((item, i) => (
          <View key={i} style={styles.bullet}>
            <Text style={[styles.bulletDot, { color: colors.tint }]}>•</Text>
            <Text style={[styles.bulletText, { color: colors.textPrimary }]}>{item}</Text>
          </View>
        ))}
        <Text style={[styles.bodyText, { color: colors.textPrimary, marginTop: 8 }]}>
          Aucune donnée n'est cédée à des tiers commerciaux.
        </Text>
      </Section>

      {/* ── 6. Vos droits ──────────────────────────────────────────────── */}
      <Section title="Vos droits" colors={colors}>
        <Text style={[styles.bodyText, { color: colors.textPrimary }]}>
          Conformément au RGPD, vous disposez des droits suivants sur vos données personnelles :
        </Text>

        {[
          { droit: 'Droit d\'accès (Art. 15)', desc: 'Obtenir une copie des données vous concernant.' },
          { droit: 'Droit de rectification (Art. 16)', desc: 'Faire corriger des données inexactes.' },
          { droit: 'Droit à l\'effacement (Art. 17)', desc: 'Demander la suppression de vos données. Exception : les données de présence et les données techniques (LCEN) sont soumises à des obligations légales de conservation ; elles ne peuvent être effacées pendant la durée applicable (Art. 17.3.b RGPD).' },
          { droit: 'Droit à la portabilité (Art. 20)', desc: 'Recevoir vos données dans un format structuré et lisible.' },
          { droit: 'Droit d\'opposition (Art. 21)', desc: 'Vous opposer à certains traitements de vos données.' },
        ].map(item => (
          <View key={item.droit} style={[styles.rightItem, { borderBottomColor: colors.dividerLine }]}>
            <Text style={[styles.rightTitle, { color: colors.textPrimary }]}>{item.droit}</Text>
            <Text style={[styles.rightDesc, { color: colors.textSecondary }]}>{item.desc}</Text>
          </View>
        ))}

        <Text style={[styles.bodyText, { color: colors.textPrimary, marginTop: 12 }]}>
          Pour exercer vos droits, contactez le DPO de l'établissement :
        </Text>
        <TouchableOpacity onPress={() => Linking.openURL('mailto:dpo@mines-ales.fr')}>
          <Text style={[styles.link, { color: colors.tint }]}>dpo@mines-ales.fr →</Text>
        </TouchableOpacity>
      </Section>

      {/* ── 7. Réclamation CNIL ────────────────────────────────────────── */}
      <Section title="Droit de réclamation" colors={colors}>
        <Text style={[styles.bodyText, { color: colors.textPrimary }]}>
          Si vous estimez que le traitement de vos données ne respecte pas la réglementation, vous avez le droit d'introduire une réclamation auprès de la Commission Nationale de l'Informatique et des Libertés (CNIL).
        </Text>
        <TouchableOpacity onPress={() => Linking.openURL('https://www.cnil.fr')}>
          <Text style={[styles.link, { color: colors.tint }]}>www.cnil.fr →</Text>
        </TouchableOpacity>
      </Section>

      {/* ── Pied de page ───────────────────────────────────────────────── */}
      <View style={[styles.footer, { borderTopColor: colors.dividerLine }]}>
        <Text style={[styles.footerText, { color: colors.textSecondary }]}>
          Dernière mise à jour : juillet 2026
        </Text>
        <Text style={[styles.footerText, { color: colors.textSecondary }]}>
          IMT Mines Alès — Tous droits réservés
        </Text>
      </View>

      {children}

    </ScrollView>
  );
}

// ─── Styles ───────────────────────────────────────────────────────────────────

export const styles = StyleSheet.create({
  scroll: {
    paddingHorizontal: 20,
    paddingBottom: 40,
  },

  // En-tête
  header: {
    alignItems: 'center',
    paddingVertical: 32,
  },
  appIcon: {
    width: 56,
    height: 56,
    borderRadius: 14,
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 12,
  },
  appIconText: {
    color: '#fff',
    fontSize: 28,
    fontWeight: '700',
  },
  appName: {
    fontSize: 22,
    fontWeight: '700',
    marginBottom: 4,
  },
  appTagline: {
    fontSize: 14,
    textAlign: 'center',
  },

  // Sections
  section: {
    marginBottom: 28,
  },
  sectionTitle: {
    fontSize: 16,
    fontWeight: '700',
    marginBottom: 12,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },

  // Lignes clé-valeur
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
    gap: 12,
  },
  rowLabel: {
    fontSize: 13,
    flex: 1,
  },
  rowValue: {
    fontSize: 13,
    flex: 2,
    textAlign: 'right',
  },

  // Texte courant
  bodyText: {
    fontSize: 14,
    lineHeight: 22,
    marginBottom: 8,
  },
  legalBase: {
    fontSize: 12,
    lineHeight: 18,
    marginTop: 12,
    fontStyle: 'italic',
  },
  link: {
    fontSize: 14,
    fontWeight: '500',
    marginTop: 8,
    marginBottom: 4,
  },

  // Liste données collectées
  dataList: {
    gap: 10,
    marginVertical: 10,
  },
  dataItem: {
    borderLeftWidth: 3,
    paddingLeft: 12,
    paddingVertical: 4,
  },
  dataCat: {
    fontSize: 13,
    fontWeight: '700',
    marginBottom: 2,
  },
  dataDetail: {
    fontSize: 13,
    lineHeight: 20,
  },

  // Boîte info LCEN
  infoBox: {
    borderLeftWidth: 4,
    borderRadius: 8,
    padding: 14,
    marginTop: 4,
  },
  infoTitle: {
    fontSize: 14,
    fontWeight: '700',
    marginBottom: 8,
  },
  infoText: {
    fontSize: 13,
    lineHeight: 20,
  },

  // Bullets
  bullet: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 8,
    marginVertical: 4,
  },
  bulletDot: {
    fontSize: 16,
    lineHeight: 22,
  },
  bulletText: {
    flex: 1,
    fontSize: 13,
    lineHeight: 20,
  },

  // Droits
  rightItem: {
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  rightTitle: {
    fontSize: 13,
    fontWeight: '600',
    marginBottom: 2,
  },
  rightDesc: {
    fontSize: 13,
    lineHeight: 19,
  },

  // Footer
  footer: {
    borderTopWidth: StyleSheet.hairlineWidth,
    paddingTop: 20,
    marginTop: 8,
    alignItems: 'center',
    gap: 4,
  },
  footerText: {
    fontSize: 12,
  },
});
