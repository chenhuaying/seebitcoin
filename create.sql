-- 数字币基本信息表，包括名称等信息
CREATE TABLE IF NOT EXISTS `cc_info` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(32) DEFAULT NULL,
  `symbol` varchar(16) DEFAULT NULL,
  `time` datetime NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- 数字币市场容量信息表
CREATE TABLE IF NOT EXISTS `cc_market` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `ccid` int(10) unsigned NOT NULL,
  `marketcap` bigint(20) NOT NULL,
  `volume_24h` bigint(20) NOT NULL,
  `circulating_supply` bigint(20) NOT NULL,
  `change_24h` float NOT NULL,
  `time` datetime NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `FK_market_ccid` FOREIGN KEY (`ccid`) REFERENCES `cc_info` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- 数字币准实时交易价格信息表
CREATE TABLE IF NOT EXISTS `cc_price` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `ccid` int(10) unsigned NOT NULL,
  `price` float NOT NULL,
  `time` datetime NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `FK_price_ccid` FOREIGN KEY (`ccid`) REFERENCES `cc_info` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
