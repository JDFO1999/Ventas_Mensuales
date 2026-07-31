USE Sistemas
GO

-- =============================================
-- Reiniciar PosVentas desde cero
-- Ejecutar UNA SOLA VEZ en 10.10.10.21
-- Luego abrir la app y procesar normalmente
-- =============================================

DROP TABLE IF EXISTS PosVentas

CREATE TABLE PosVentas (
    Fecha      DATE NOT NULL,
    Hora       TINYINT NOT NULL,
    Tienda     VARCHAR(6) NOT NULL,
    Caja       INT NOT NULL,
    Tipo       CHAR(2) NOT NULL,
    STipo      CHAR(2) NULL,
    Numero     VARCHAR(15) NOT NULL,
    Anulada    CHAR(1) NULL,
    Operador   VARCHAR(20) NULL,
    SubTotal   DECIMAL(18,2) NULL,
    MontoUSD   DECIMAL(18,2) NOT NULL,
    MontoBS    DECIMAL(18,2) NULL,
    Impuesto   DECIMAL(18,2) NULL,
    Descuento  DECIMAL(18,2) NULL,
    IGTF       DECIMAL(18,2) NULL,
    Redondeo   DECIMAL(18,2) NULL,
    TasaDOL    DECIMAL(20,6) NULL,
    CodCli     VARCHAR(10) NULL,
    Nombres    VARCHAR(100) NULL,
    Apellidos  VARCHAR(100) NULL,
    NIT        VARCHAR(14) NULL,
    RIF        VARCHAR(14) NULL,
    CodVen     VARCHAR(2) NULL,
    NroZ       VARCHAR(10) NULL,
    Credito    BIT NULL,
    Vuelto     DECIMAL(16,2) NULL,
    LIMPRIMIO  BIT NULL,
    NroCie     VARCHAR(8) NULL,
    FechaCie   DATE NULL,
    FechaCarga DATETIME DEFAULT GETDATE(),
    CONSTRAINT PK_PosVentas PRIMARY KEY (Tienda, Caja, Numero, Tipo)
)

CREATE NONCLUSTERED INDEX IX_PosVentas_Fecha ON PosVentas(Tienda, Fecha)

PRINT 'PosVentas recreada con PK e indice'

-- =============================================
-- Indice para H_Tiendas (FA y CA)
-- =============================================
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE object_id = OBJECT_ID('H_Tiendas') AND name = 'IX_H_Tiendas_Busqueda')
    CREATE NONCLUSTERED INDEX IX_H_Tiendas_Busqueda ON H_Tiendas(Tienda, Anio, Mes)

PRINT 'Indice H_Tiendas listo'
GO
