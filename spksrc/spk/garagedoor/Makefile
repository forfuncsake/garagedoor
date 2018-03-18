SPK_NAME = garagedoor
SPK_VERS = 0.1.0
SPK_REV = 1

include ../../mk/spksrc.common.mk

MAINTAINER = forfuncsake
DESCRIPTION = Runs a homekit proxy for DIY smart garage door controller
RELOAD_UI = no
DISPLAY_NAME = GDHK

HOMEPAGE = 
LICENSE  =

INSTALLER_SCRIPT = src/installer.sh
SSS_SCRIPT       = src/dsm-control.sh

INSTALL_PREFIX = /usr/local/$(SPK_NAME)

DEPEND_TARGET = nop
COPY_TARGET = gdhk_extra_install
STRIP_TARGET = nop
WHEEL_TARGET = nop

include ../../mk/spksrc.spk.mk

.PHONY: gdhk_extra_install
gdhk_extra_install:
	install -m 755 -d $(STAGING_DIR)/bin
	install -m 755 -d $(STAGING_DIR)/var
	install -m 755 -d $(STAGING_DIR)/var/log
	cp src/gdhk $(STAGING_DIR)/bin/
