-- Fix one-time: reconstruye H_Tiendas con Id IDENTITY(1,1) y esquema correcto.
-- Conserva las filas existentes y sus Id. Requiere ser ejecutado por un usuario
-- con permisos de DDL sobre la base Sistemas (dbo).
-- El exe (CrearTablaH_Tiendas) hace esto automaticamente al iniciar; este script
-- es solo por si se prefiere corregir sin recompilar/desplegar.

USE Sistemas;
GO

IF (SELECT COLUMNPROPERTY(OBJECT_ID('H_Tiendas'),'Id','IsIdentity')) = 0
BEGIN
	IF OBJECT_ID('dbo.H_Tiendas_New','U') IS NOT NULL
	DROP TABLE dbo.H_Tiendas_New;

	CREATE TABLE dbo.H_Tiendas_New (
		Id              INT IDENTITY(1,1) PRIMARY KEY,
		FechaEjecucion  DATETIME NOT NULL DEFAULT GETDATE(),
		Tienda          VARCHAR(6) NOT NULL,
		Anio            INT NOT NULL,
		Mes             INT NOT NULL,
		RegistrosDBF    INT NOT NULL DEFAULT 0,
		RegistrosSQL    INT NOT NULL DEFAULT 0,
		Insertados      INT NOT NULL DEFAULT 0,
		Estado          VARCHAR(20) NOT NULL,
		DuracionSeg     INT NULL,
		Modo            VARCHAR(10) NOT NULL,
		TipoEjecucion   VARCHAR(20) NOT NULL,
		Sistema         VARCHAR(10) NULL
	);

	SET IDENTITY_INSERT dbo.H_Tiendas_New ON;
	INSERT INTO dbo.H_Tiendas_New (Id, FechaEjecucion, Tienda, Anio, Mes, RegistrosDBF, RegistrosSQL, Insertados, Estado, DuracionSeg, Modo, TipoEjecucion, Sistema)
	SELECT Id, FechaEjecucion, Tienda, Anio, Mes, RegistrosDBF, RegistrosSQL, Insertados, Estado, DuracionSeg, Modo, TipoEjecucion, Sistema FROM dbo.H_Tiendas;
	SET IDENTITY_INSERT dbo.H_Tiendas_New OFF;

	DROP TABLE dbo.H_Tiendas;
	EXEC sp_rename N'dbo.H_Tiendas_New', N'H_Tiendas';

	IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('H_Tiendas') AND name = 'IX_H_Tiendas_Busqueda')
	CREATE NONCLUSTERED INDEX IX_H_Tiendas_Busqueda ON H_Tiendas(Tienda, Anio, Mes);

	PRINT 'H_Tiendas reconstruida con Id IDENTITY.';
END
ELSE
BEGIN
	PRINT 'H_Tiendas ya tiene Id IDENTITY; no se requiere accion.';
END
GO
