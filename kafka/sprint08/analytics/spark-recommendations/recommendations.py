import argparse

from pyspark.sql import SparkSession
from pyspark.sql import functions as F
from pyspark.sql.window import Window


def parse_args():
    parser = argparse.ArgumentParser(description="Calculate marketplace recommendations from HDFS and publish them to Kafka")
    parser.add_argument("--input", default="hdfs://namenode:8020/marketplace-analytics/products_allowed/*/*.json")
    parser.add_argument("--output", default="hdfs://namenode:8020/marketplace-analytics/spark_recommendations")
    parser.add_argument("--bootstrap-servers", default="kafka2-1:29092,kafka2-2:29092,kafka2-3:29092")
    parser.add_argument("--topic", default="analytics.recommendations")
    parser.add_argument("--truststore", default="/certs/kafka.truststore.jks")
    parser.add_argument("--keystore", default="/certs/admin.keystore.jks")
    parser.add_argument("--password", default="changeit")
    return parser.parse_args()


def main():
    args = parse_args()
    spark = (
        SparkSession.builder.appName("marketplace-spark-recommendations")
        .getOrCreate()
    )

    raw = spark.read.json(args.input)
    products = raw.select(
        F.col("payload.product_id").alias("product_id"),
        F.col("payload.name").alias("name"),
        F.col("payload.category").alias("category"),
        F.col("payload.stock.available").cast("double").alias("available"),
        F.col("payload.stock.reserved").cast("double").alias("reserved"),
        F.col("payload.updated_at").alias("updated_at"),
    ).where(F.col("product_id").isNotNull() & F.col("category").isNotNull())

    scored = products.withColumn(
        "score",
        F.greatest((F.col("available") - F.col("reserved")) / F.lit(100.0), F.lit(0.0)),
    )

    window = Window.partitionBy("category").orderBy(F.col("score").desc(), F.col("updated_at").desc())
    top_products = scored.withColumn("rn", F.row_number().over(window)).where(F.col("rn") <= 5)

    recommendations = top_products.groupBy("category").agg(
        F.collect_list(
            F.struct(
                F.col("rn").alias("rank"),
                F.col("product_id"),
                F.col("name"),
                F.round(F.col("score"), 4).alias("score"),
            )
        ).alias("ranked_products")
    ).select(
        F.col("category"),
        F.expr("uuid()").alias("event_id"),
        F.expr("uuid()").alias("recommendation_id"),
        F.current_timestamp().alias("calculated_at"),
        F.expr("transform(array_sort(ranked_products), x -> named_struct('product_id', x.product_id, 'name', x.name, 'score', x.score))").alias("products"),
    )

    events = recommendations.select(
        F.col("category").alias("key"),
        F.struct(
            F.col("event_id"),
            F.lit("recommendations_calculated").alias("event_type"),
            F.col("calculated_at").alias("event_time"),
            F.lit("spark-recommendations").alias("source"),
            F.struct(
                F.col("recommendation_id"),
                F.col("category"),
                F.col("products"),
                F.col("calculated_at"),
            ).alias("payload"),
        ).alias("value"),
    )

    events.select("value.*").write.mode("overwrite").json(args.output)

    events.select(F.col("key"), F.to_json(F.col("value")).alias("value")).selectExpr("CAST(key AS STRING)", "CAST(value AS STRING)").write.format("kafka").option(
        "kafka.bootstrap.servers", args.bootstrap_servers
    ).option("topic", args.topic).option("kafka.security.protocol", "SSL").option(
        "kafka.ssl.truststore.location", args.truststore
    ).option("kafka.ssl.truststore.password", args.password).option(
        "kafka.ssl.truststore.type", "PKCS12"
    ).option("kafka.ssl.keystore.location", args.keystore).option(
        "kafka.ssl.keystore.password", args.password
    ).option("kafka.ssl.keystore.type", "PKCS12").option(
        "kafka.ssl.key.password", args.password
    ).save()

    spark.stop()


if __name__ == "__main__":
    main()
