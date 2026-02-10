# typed: strict
# frozen_string_literal: true

class MaintenanceMiddleware
  extend T::Sig

  HEALTH_CHECK_PATH = T.let("/health", String)

  sig { params(app: T.untyped).void }
  def initialize(app)
    @app = T.let(app, T.untyped)
  end

  sig do
    params(
      env: T::Hash[String, T.untyped]
    ).returns(T.untyped)
  end
  def call(env)
    unless maintenance_mode?
      return @app.call(env)
    end

    if health_check?(env)
      return @app.call(env)
    end

    if admin_ip?(env)
      return @app.call(env)
    end

    maintenance_response
  end

  sig { returns(T::Boolean) }
  private def maintenance_mode?
    ENV["WIKINO_MAINTENANCE_MODE"] == "on"
  end

  sig do
    params(
      env: T::Hash[String, T.untyped]
    ).returns(T::Boolean)
  end
  private def admin_ip?(env)
    ip = client_ip(env)
    admin_ips.include?(ip)
  end

  sig do
    params(
      env: T::Hash[String, T.untyped]
    ).returns(T::Boolean)
  end
  private def health_check?(env)
    env["PATH_INFO"] == HEALTH_CHECK_PATH
  end

  sig do
    params(
      env: T::Hash[String, T.untyped]
    ).returns(String)
  end
  private def client_ip(env)
    env["HTTP_CF_CONNECTING_IP"] ||
      forwarded_ip(env) ||
      env["HTTP_X_REAL_IP"] ||
      env["REMOTE_ADDR"] ||
      ""
  end

  sig do
    params(
      env: T::Hash[String, T.untyped]
    ).returns(T.nilable(String))
  end
  private def forwarded_ip(env)
    value = env["HTTP_X_FORWARDED_FOR"]

    unless value
      return nil
    end

    value.split(",").first&.strip
  end

  sig { returns(T::Array[String]) }
  private def admin_ips
    ENV
      .fetch("WIKINO_ADMIN_IP", "")
      .split(",")
      .map(&:strip)
      .reject(&:empty?)
  end

  sig { returns(T::Array[T.untyped]) }
  private def maintenance_response
    html_path = Rails.public_path.join(
      "maintenance.html"
    )

    body = if html_path.exist?
      html_path.read
    else
      "<html><body><h1>メンテナンス中</h1></body></html>"
    end

    retry_after = (Time.now.utc + 3600).httpdate

    [
      503,
      {
        "content-type" => "text/html; charset=utf-8",
        "retry-after" => retry_after
      },
      [body]
    ]
  end
end
