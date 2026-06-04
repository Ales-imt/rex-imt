-- MariaDB dump 10.19  Distrib 10.7.3-MariaDB, for debian-linux-gnu (x86_64)
--
-- Host: localhost    Database: cyber_notes_v2
-- ------------------------------------------------------
-- Server version	10.7.3-MariaDB-1:10.7.3+maria~focal

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `DetailEleve`
--

DROP TABLE IF EXISTS `DetailEleve`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `DetailEleve` (
  `IDDetail` int(11) NOT NULL AUTO_INCREMENT,
  `dtn` date DEFAULT NULL,
  `adresse` varchar(200) DEFAULT NULL,
  `tel` varchar(16) DEFAULT NULL,
  `fax` varchar(16) DEFAULT NULL,
  `origine` varchar(40) DEFAULT NULL,
  `EVCLEUNIK` int(11) DEFAULT 0,
  `lieu_n` varchar(50) DEFAULT NULL,
  `dep_n` varchar(50) DEFAULT NULL,
  `natio` varchar(50) DEFAULT NULL,
  `Bac` varchar(5) DEFAULT NULL,
  `An_Bac` smallint(5) unsigned DEFAULT 0,
  `M_Bac` varchar(20) DEFAULT NULL,
  `lycee_origine` varchar(70) DEFAULT NULL,
  `Ac_Bac` varchar(70) DEFAULT NULL,
  `Nom_P` varchar(30) DEFAULT NULL,
  `Prenom_P` varchar(30) DEFAULT NULL,
  `Prof_P` varchar(50) DEFAULT NULL,
  `Cat_P` varchar(2) DEFAULT NULL,
  `Tel_P` varchar(20) DEFAULT NULL,
  `Port_P` varchar(20) DEFAULT NULL,
  `Mail_P` varchar(50) DEFAULT NULL,
  `Nom_M` varchar(30) DEFAULT NULL,
  `Prenom_M` varchar(30) DEFAULT NULL,
  `Adresse_P` varchar(200) DEFAULT NULL,
  `Prof_M` varchar(50) DEFAULT NULL,
  `Cat_M` varchar(2) DEFAULT NULL,
  `Adresse_M` varchar(200) DEFAULT NULL,
  `Tel_M` varchar(20) DEFAULT NULL,
  `Port_M` varchar(20) DEFAULT NULL,
  `Mail_M` varchar(50) DEFAULT NULL,
  `Coord_HPS` varchar(200) DEFAULT NULL,
  `Coord_Urg` varchar(200) DEFAULT NULL,
  `photo` longblob DEFAULT NULL,
  `Ville_lycee` varchar(50) DEFAULT NULL,
  `pays_n` varchar(50) DEFAULT NULL,
  `tous_prenoms` varchar(200) DEFAULT NULL,
  PRIMARY KEY (`IDDetail`) USING BTREE,
  UNIQUE KEY `EVCLEUNIK` (`EVCLEUNIK`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=8721 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `Exercice`
--

DROP TABLE IF EXISTS `Exercice`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `Exercice` (
  `CTCLEUNIK` int(11) NOT NULL AUTO_INCREMENT,
  `nom` varchar(150) DEFAULT NULL,
  `date` date DEFAULT NULL,
  `base` smallint(6) DEFAULT 0,
  `coefficient` float DEFAULT 0,
  `P0CLEUNIK` int(11) DEFAULT 0,
  `IDTypeExercice` int(11) DEFAULT 0,
  `Supplement` tinyint(4) DEFAULT 0,
  `DureeEx` int(11) DEFAULT 0,
  `Commentaire` longtext DEFAULT NULL,
  PRIMARY KEY (`CTCLEUNIK`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=19198 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `Niveau`
--

DROP TABLE IF EXISTS `Niveau`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `Niveau` (
  `IDNiveau` int(11) NOT NULL AUTO_INCREMENT,
  `nom` varchar(40) DEFAULT NULL,
  `description` longtext DEFAULT NULL,
  `couleurFond` bigint(20) DEFAULT 0,
  `couleurTexte` bigint(20) DEFAULT 0,
  `password` varchar(20) DEFAULT NULL,
  `colori` smallint(5) unsigned DEFAULT 0,
  `p0encours` int(11) DEFAULT 0,
  PRIMARY KEY (`IDNiveau`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=48 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `Promos_eleves`
--

DROP TABLE IF EXISTS `Promos_eleves`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `Promos_eleves` (
  `EVCLEUNIK` int(11) DEFAULT 0,
  `P0CLEUNIK` int(11) DEFAULT 0,
  `Etat` int(11) DEFAULT 0,
  UNIQUE KEY `IDpromos_eleves` (`P0CLEUNIK`,`EVCLEUNIK`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `Realisation`
--

DROP TABLE IF EXISTS `Realisation`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `Realisation` (
  `NOCLEUNIK` int(11) NOT NULL AUTO_INCREMENT,
  `base` smallint(6) DEFAULT 0,
  `coefficient` float DEFAULT 0,
  `date` date DEFAULT NULL,
  `EVCLEUNIK` int(11) DEFAULT 0,
  `CTCLEUNIK` int(11) DEFAULT 0,
  `noteobtenue` float DEFAULT -1,
  `Nom_etablissement` varchar(50) DEFAULT NULL,
  `Adresse_etablissement` varchar(300) DEFAULT NULL,
  `Ville` varchar(50) DEFAULT NULL,
  `Pays` varchar(50) DEFAULT NULL,
  `Sujet` varchar(300) DEFAULT NULL,
  `Tuteur` varchar(50) DEFAULT NULL,
  `Commentaire` longtext DEFAULT NULL,
  `cp` bigint(20) DEFAULT 0,
  `duree` int(11) DEFAULT 0,
  PRIMARY KEY (`NOCLEUNIK`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=847203 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `TypeExercice`
--

DROP TABLE IF EXISTS `TypeExercice`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `TypeExercice` (
  `IDTypeExercice` int(11) NOT NULL AUTO_INCREMENT,
  `nom` varchar(40) DEFAULT NULL,
  `description` longtext DEFAULT NULL,
  `Supplement` tinyint(4) DEFAULT 0,
  `Aff_haut` tinyint(3) unsigned DEFAULT 0,
  `Aff_bas` tinyint(3) unsigned DEFAULT 0,
  PRIMARY KEY (`IDTypeExercice`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `eleves`
--

DROP TABLE IF EXISTS `eleves`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `eleves` (
  `EVCLEUNIK` int(11) NOT NULL AUTO_INCREMENT,
  `det` varchar(5) DEFAULT NULL,
  `nom` varchar(40) DEFAULT NULL,
  `prenom` varchar(40) DEFAULT NULL,
  `mel` varchar(80) DEFAULT NULL,
  `type` varchar(40) DEFAULT NULL,
  `password` varchar(150) DEFAULT NULL,
  `typepassword` varchar(30) DEFAULT NULL,
  `INE` varchar(15) DEFAULT '0',
  PRIMARY KEY (`EVCLEUNIK`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=8730 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `promos`
--

DROP TABLE IF EXISTS `promos`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `promos` (
  `P0CLEUNIK` int(11) NOT NULL AUTO_INCREMENT,
  `nom` varchar(40) DEFAULT NULL,
  `afficher` tinyint(4) DEFAULT 0,
  `datedebut` date DEFAULT NULL,
  `datefin` date DEFAULT NULL,
  `IDNiveau` int(11) DEFAULT 0,
  `dateImpressionDiffusee` date DEFAULT NULL,
  `liensSupDiplome` longtext DEFAULT NULL,
  `texteSupDiplome` longtext DEFAULT NULL,
  PRIMARY KEY (`P0CLEUNIK`) USING BTREE,
  UNIQUE KEY `nom` (`nom`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=1147 DEFAULT CHARSET=utf8mb4;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-06-04  8:07:20
