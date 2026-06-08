/*M!999999\- enable the sandbox mode */ 
-- MariaDB dump 10.19-12.3.2-MariaDB, for debian-linux-gnu (x86_64)
--
-- Host: sql2.mines-ales.fr    Database: cybernotes
-- ------------------------------------------------------
-- Server version	5.7.24

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*M!100616 SET @OLD_NOTE_VERBOSITY=@@NOTE_VERBOSITY, NOTE_VERBOSITY=0 */;

--
-- Table structure for table `DetailEleve`
--

DROP TABLE IF EXISTS `DetailEleve`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `DetailEleve` (
  `IDDetail` int(11) NOT NULL AUTO_INCREMENT,
  `dtn` date DEFAULT NULL,
  `adresse` varchar(200) DEFAULT '',
  `tel` varchar(16) DEFAULT '',
  `fax` varchar(16) DEFAULT '',
  `origine` varchar(40) DEFAULT '',
  `EVCLEUNIK` int(11) DEFAULT '0',
  `lieu_n` varchar(50) DEFAULT '',
  `dep_n` varchar(50) DEFAULT '',
  `natio` varchar(50) DEFAULT '',
  `Bac` varchar(5) DEFAULT '',
  `An_Bac` smallint(5) unsigned DEFAULT '0',
  `M_Bac` varchar(20) DEFAULT '',
  `lycee_origine` varchar(70) DEFAULT '',
  `Ac_Bac` varchar(70) DEFAULT '',
  `Nom_P` varchar(30) DEFAULT '',
  `Prenom_P` varchar(30) DEFAULT '',
  `Prof_P` varchar(50) DEFAULT '',
  `Cat_P` varchar(2) DEFAULT '',
  `Tel_P` varchar(20) DEFAULT '',
  `Port_P` varchar(20) DEFAULT '',
  `Mail_P` varchar(50) DEFAULT '',
  `Nom_M` varchar(30) DEFAULT '',
  `Prenom_M` varchar(30) DEFAULT '',
  `Adresse_P` varchar(200) DEFAULT '',
  `Prof_M` varchar(50) DEFAULT '',
  `Cat_M` varchar(2) DEFAULT '',
  `Adresse_M` varchar(200) DEFAULT '',
  `Tel_M` varchar(20) DEFAULT '',
  `Port_M` varchar(20) DEFAULT '',
  `Mail_M` varchar(50) DEFAULT '',
  `Coord_HPS` varchar(200) DEFAULT '',
  `Coord_Urg` varchar(200) DEFAULT '',
  `photo` longblob,
  `Ville_lycee` varchar(50) DEFAULT '',
  `pays_n` varchar(50) DEFAULT '',
  `tous_prenoms` varchar(200) DEFAULT '',
  PRIMARY KEY (`IDDetail`),
  UNIQUE KEY `EVCLEUNIK` (`EVCLEUNIK`)
) ENGINE=InnoDB AUTO_INCREMENT=8721 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `Exercice`
--

DROP TABLE IF EXISTS `Exercice`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `Exercice` (
  `CTCLEUNIK` int(11) NOT NULL AUTO_INCREMENT,
  `nom` varchar(150) DEFAULT '',
  `date` date DEFAULT NULL,
  `base` smallint(6) DEFAULT '0',
  `coefficient` float DEFAULT '0',
  `P0CLEUNIK` int(11) DEFAULT '0',
  `IDTypeExercice` int(11) DEFAULT '0',
  `Supplement` tinyint(4) DEFAULT '0',
  `DureeEx` int(11) DEFAULT '0',
  `Commentaire` longtext,
  PRIMARY KEY (`CTCLEUNIK`),
  KEY `WDIDX16796532040` (`date`),
  KEY `WDIDX16796532041` (`P0CLEUNIK`),
  KEY `WDIDX16796532042` (`IDTypeExercice`)
) ENGINE=InnoDB AUTO_INCREMENT=19362 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `Niveau`
--

DROP TABLE IF EXISTS `Niveau`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `Niveau` (
  `IDNiveau` int(11) NOT NULL AUTO_INCREMENT,
  `nom` varchar(40) DEFAULT '',
  `description` longtext,
  `couleurFond` bigint(20) DEFAULT '0',
  `couleurTexte` bigint(20) DEFAULT '0',
  `password` varchar(20) DEFAULT '',
  `colori` smallint(5) unsigned DEFAULT '0',
  `p0encours` int(11) DEFAULT '0',
  PRIMARY KEY (`IDNiveau`),
  KEY `WDIDX167965320510` (`nom`)
) ENGINE=InnoDB AUTO_INCREMENT=48 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `Promos_eleves`
--

DROP TABLE IF EXISTS `Promos_eleves`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `Promos_eleves` (
  `EVCLEUNIK` int(11) DEFAULT '0',
  `P0CLEUNIK` int(11) DEFAULT '0',
  `Etat` int(11) DEFAULT '0',
  UNIQUE KEY `IDpromos_eleves` (`P0CLEUNIK`,`EVCLEUNIK`),
  KEY `WDIDX167965320511` (`EVCLEUNIK`),
  KEY `WDIDX167965320512` (`P0CLEUNIK`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `Realisation`
--

DROP TABLE IF EXISTS `Realisation`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `Realisation` (
  `NOCLEUNIK` int(11) NOT NULL AUTO_INCREMENT,
  `base` smallint(6) DEFAULT '0',
  `coefficient` float DEFAULT '0',
  `date` date DEFAULT NULL,
  `EVCLEUNIK` int(11) DEFAULT '0',
  `CTCLEUNIK` int(11) DEFAULT '0',
  `noteobtenue` float DEFAULT '-1',
  `Nom_etablissement` varchar(50) DEFAULT '',
  `Adresse_etablissement` varchar(300) DEFAULT '',
  `Ville` varchar(50) DEFAULT '',
  `Pays` varchar(50) DEFAULT '',
  `Sujet` varchar(300) DEFAULT '',
  `Tuteur` varchar(50) DEFAULT '',
  `Commentaire` longtext,
  `cp` bigint(20) DEFAULT '0',
  `duree` int(11) DEFAULT '0',
  PRIMARY KEY (`NOCLEUNIK`),
  KEY `WDIDX16796532057` (`EVCLEUNIK`),
  KEY `WDIDX16796532058` (`CTCLEUNIK`)
) ENGINE=InnoDB AUTO_INCREMENT=855682 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `TypeExercice`
--

DROP TABLE IF EXISTS `TypeExercice`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `TypeExercice` (
  `IDTypeExercice` int(11) NOT NULL AUTO_INCREMENT,
  `nom` varchar(40) DEFAULT '',
  `description` longtext,
  `Supplement` tinyint(4) DEFAULT '0',
  `Aff_haut` tinyint(3) unsigned DEFAULT '0',
  `Aff_bas` tinyint(3) unsigned DEFAULT '0',
  PRIMARY KEY (`IDTypeExercice`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `eleves`
--

DROP TABLE IF EXISTS `eleves`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `eleves` (
  `EVCLEUNIK` int(11) NOT NULL AUTO_INCREMENT,
  `det` varchar(5) DEFAULT '',
  `nom` varchar(40) DEFAULT '',
  `prenom` varchar(40) DEFAULT '',
  `mel` varchar(80) DEFAULT '',
  `type` varchar(40) DEFAULT '',
  `password` varchar(150) DEFAULT '',
  `typepassword` varchar(30) DEFAULT '',
  `INE` varchar(15) DEFAULT '0',
  PRIMARY KEY (`EVCLEUNIK`),
  KEY `WDIDX16796532053` (`nom`),
  KEY `WDIDX16796532054` (`mel`),
  KEY `WDIDX16796532055` (`password`),
  KEY `WDIDX16796532056` (`INE`)
) ENGINE=InnoDB AUTO_INCREMENT=8730 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `promos`
--

DROP TABLE IF EXISTS `promos`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `promos` (
  `P0CLEUNIK` int(11) NOT NULL AUTO_INCREMENT,
  `nom` varchar(40) DEFAULT '',
  `afficher` tinyint(4) DEFAULT '0',
  `datedebut` date DEFAULT NULL,
  `datefin` date DEFAULT NULL,
  `IDNiveau` int(11) DEFAULT '0',
  `dateImpressionDiffusee` date DEFAULT NULL,
  `liensSupDiplome` longtext,
  `texteSupDiplome` longtext,
  PRIMARY KEY (`P0CLEUNIK`),
  UNIQUE KEY `nom` (`nom`),
  KEY `WDIDX16796532059` (`IDNiveau`)
) ENGINE=InnoDB AUTO_INCREMENT=1148 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*M!100616 SET NOTE_VERBOSITY=@OLD_NOTE_VERBOSITY */;

-- Dump completed on 2026-06-08  6:50:58
