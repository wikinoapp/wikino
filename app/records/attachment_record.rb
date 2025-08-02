# typed: strict
# frozen_string_literal: true

require "sanitize"

class AttachmentRecord < ApplicationRecord
  # SVGファイルで許可する要素
  SVG_ALLOWED_ELEMENTS = T.let(
    %w[
      svg g path rect circle ellipse line polyline polygon text tspan
      defs clipPath mask pattern linearGradient radialGradient stop
      use symbol marker image title desc metadata
    ].freeze,
    T::Array[String]
  )

  # SVGファイルで許可する属性
  SVG_ALLOWED_ATTRIBUTES = T.let(
    {
      :all => %w[id class style fill stroke stroke-width stroke-linecap
        stroke-linejoin stroke-miterlimit stroke-dasharray
        stroke-dashoffset stroke-opacity fill-opacity opacity
        transform],
      "svg" => %w[xmlns xmlns:xlink viewBox width height preserveAspectRatio],
      "image" => %w[x y width height xlink:href href preserveAspectRatio],
      "a" => %w[href xlink:href target],
      "text" => %w[x y dx dy text-anchor font-family font-size font-weight],
      "tspan" => %w[x y dx dy],
      "linearGradient" => %w[x1 y1 x2 y2 gradientUnits gradientTransform],
      "radialGradient" => %w[cx cy r fx fy gradientUnits gradientTransform],
      "stop" => %w[offset stop-color stop-opacity],
      "pattern" => %w[x y width height patternUnits patternContentUnits
        patternTransform],
      "clipPath" => %w[clipPathUnits],
      "mask" => %w[x y width height maskUnits maskContentUnits],
      "use" => %w[x y width height xlink:href href],
      "path" => %w[d],
      "rect" => %w[x y width height rx ry],
      "circle" => %w[cx cy r],
      "ellipse" => %w[cx cy rx ry],
      "line" => %w[x1 y1 x2 y2],
      "polyline" => %w[points],
      "polygon" => %w[points]
    }.freeze,
    T::Hash[T.any(String, Symbol), T::Array[String]]
  )

  # 危険なプロトコルのパターン
  DANGEROUS_PROTOCOLS = T.let(
    /\A(javascript|data|vbscript):/i,
    Regexp
  )

  self.table_name = "attachments"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :attached_space_member_record,
    class_name: "SpaceMemberRecord",
    foreign_key: :attached_space_member_id
  belongs_to :active_storage_attachment_record,
    class_name: "ActiveStorage::Attachment"

  has_many :page_attachment_reference_records,
    class_name: "PageAttachmentReferenceRecord",
    foreign_key: :attachment_id,
    dependent: :restrict_with_exception

  scope :by_space, ->(space_id) { where(space_id:) }
  scope :by_space_member, ->(attached_space_member_id) { where(attached_space_member_id:) }
  scope :recent, -> { order(attached_at: :desc) }

  # Active Storageのblobへのショートカット
  sig { returns(T.nilable(ActiveStorage::Blob)) }
  def blob_record
    active_storage_attachment_record&.blob
  end

  # ファイル名を取得
  sig { returns(T.nilable(String)) }
  def filename
    blob_record&.filename&.to_s
  end

  # コンテントタイプを取得
  sig { returns(T.nilable(String)) }
  def content_type
    blob_record&.content_type
  end

  # ファイルサイズを取得（バイト単位）
  sig { returns(T.nilable(Integer)) }
  def byte_size
    blob_record&.byte_size
  end

  # SVGファイルのサニタイズ処理
  sig { returns(T::Boolean) }
  def sanitize_svg_content
    blob = blob_record
    return false unless blob
    return false unless blob.content_type == "image/svg+xml"

    begin
      # ファイルの内容を取得
      svg_content = blob.download

      # サニタイズ処理
      sanitized_content = sanitize_svg(svg_content)

      # サニタイズ済みのコンテンツをアップロード
      blob.upload(StringIO.new(sanitized_content))

      true
    rescue => e
      Rails.logger.error("SVG sanitization failed: #{e.message}")
      false
    end
  end

  sig { params(svg_content: String).returns(String) }
  private def sanitize_svg(svg_content)
    config = Sanitize::Config.merge(
      Sanitize::Config::RELAXED,
      {
        elements: SVG_ALLOWED_ELEMENTS,
        attributes: SVG_ALLOWED_ATTRIBUTES,
        protocols: {
          "a" => {"href" => ["http", "https"]},
          "image" => {"href" => ["http", "https"], "xlink:href" => ["http", "https"]}
        },
        remove_contents: %w[script style],
        remove_empty_elements: false,
        whitespace: :remove
      }
    )

    # Sanitizeを使用してSVGをクリーニング
    sanitized = Sanitize.fragment(svg_content, config)

    # 追加のセキュリティチェック
    sanitized = remove_dangerous_attributes(sanitized)
    remove_external_references(sanitized)
  end

  sig { params(content: String).returns(String) }
  private def remove_dangerous_attributes(content)
    # on*イベントハンドラを削除
    content.gsub(/\s+on\w+\s*=\s*["'][^"']*["']/i, "")
  end

  sig { params(content: String).returns(String) }
  private def remove_external_references(content)
    # 危険なプロトコルを含むhref属性を削除
    content.gsub(/href\s*=\s*["']([^"']*)[^"']/i) do |match|
      url = ::Regexp.last_match(1)
      if DANGEROUS_PROTOCOLS.match?(url)
        "" # 危険なURLは完全に削除
      else
        match
      end
    end
  end
end
