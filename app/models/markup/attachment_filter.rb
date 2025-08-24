# typed: strict
# frozen_string_literal: true

class Markup
  class AttachmentFilter < HTMLPipeline::NodeFilter
    extend T::Sig

    # インライン表示可能な画像フォーマット
    INLINE_IMAGE_FORMATS = T.let(
      %w[jpg jpeg png gif svg webp].freeze,
      T::Array[String]
    )

    # インライン表示可能な動画フォーマット
    INLINE_VIDEO_FORMATS = T.let(
      %w[mp4 webm ogg mov].freeze,
      T::Array[String]
    )

    # @override
    sig { params(context: T::Hash[Symbol, T.untyped], result: T::Hash[Symbol, T.untyped]).void }
    def initialize(context: {}, result: {})
      super
      @current_space = T.let(context[:current_space], Space)
      @current_space_member = T.let(context[:current_space_member], T.nilable(SpaceMember))
    end

    # @override
    sig { void }
    def validate
      needs(:current_space)
    end

    # @override
    sig { returns(Selma::Selector) }
    def selector
      # img要素とa要素（添付ファイルへのリンク）を対象にする
      Selma::Selector.new(match_element: "img, a")
    end

    # @override
    sig { params(element: Selma::HTML::Element).void }
    def handle_element(element)
      case element.tag_name
      when "img"
        handle_image(element)
      when "a"
        handle_link(element)
      end
    end

    sig { params(element: Selma::HTML::Element).void }
    private def handle_image(element)
      src = element["src"]
      return if src.blank?

      # 添付ファイルIDを抽出
      attachment_id = extract_attachment_id(src)
      return unless attachment_id

      # AttachmentRecordを取得してファイル形式を確認
      space_record = SpaceRecord.find_by!(identifier: current_space.identifier)
      attachment = AttachmentRecord.find_by(id: attachment_id, space_record:)
      return unless attachment

      # インライン表示可能な画像形式かチェック
      if can_display_inline_image?(attachment)
        # 元のwidth/height属性を取得
        width_attr = element["width"]
        height_attr = element["height"]

        # ファイル名を取得（alt属性用）
        filename = attachment.filename || "Image"
        escaped_filename = CGI.escapeHTML(filename)

        # プレースホルダーとして表示（署名付きURLは後でJavaScriptで置換）
        img_attrs = [
          "src=\"\"",
          "data-attachment-id=\"#{attachment_id}\"",
          "data-attachment-type=\"image\"",
          "alt=\"#{escaped_filename}\"",
          "class=\"max-w-full\""
        ]
        img_attrs << "width=\"#{width_attr}\"" if width_attr
        img_attrs << "height=\"#{height_attr}\"" if height_attr

        # a要素で囲む（href属性も後でJavaScriptで設定）
        link_html = <<~HTML
          <a href="#" data-attachment-id="#{attachment_id}" data-attachment-link="true" target="_blank" rel="noopener noreferrer" class="inline-block wikino-attachment-link">
            <img #{img_attrs.join(" ")} />
          </a>
        HTML

        # element.replaceの代わりに、after挿入してから元のelementを削除
        element.after(link_html, as: :html)
        element.remove
      else
        # インライン表示不可の場合はリンクに変換
        convert_to_download_link(element, attachment)
      end
    end

    sig { params(element: Selma::HTML::Element).void }
    private def handle_link(element)
      href = element["href"]
      return if href.blank?

      # 添付ファイルIDを抽出
      attachment_id = extract_attachment_id(href)
      return unless attachment_id

      # AttachmentRecordを取得してファイル形式を確認
      space_record = SpaceRecord.find_by!(identifier: current_space.identifier)
      attachment = AttachmentRecord.find_by(id: attachment_id, space_record:)
      return unless attachment

      # 動画ファイルの場合はvideo要素に変換
      if can_display_inline_video?(attachment)
        # video要素として表示（署名付きURLは後でJavaScriptで置換）
        video_html = <<~HTML
          <video src="" data-attachment-id="#{attachment_id}" data-attachment-type="video" controls class="max-w-full">
            お使いのブラウザは動画タグをサポートしていません。
            <a href="#" data-attachment-id="#{attachment_id}" data-attachment-link="true" target="_blank">動画をダウンロード</a>
          </video>
        HTML

        # element.replaceの代わりに、after挿入してから元のelementを削除
        element.after(video_html, as: :html)
        element.remove
      else
        # 動画以外の場合は通常のリンクとして処理
        # data属性を追加（署名付きURLは後でJavaScriptで置換）
        element["href"] = "#"
        element["data-attachment-id"] = attachment_id
        element["data-attachment-link"] = "true"

        # 新規タブで開く
        element["target"] = "_blank"
        element["rel"] = "noopener noreferrer"
      end
    end

    # 添付ファイルのURLから添付ファイルIDを抽出
    sig { params(url: String).returns(T.nilable(String)) }
    private def extract_attachment_id(url)
      return nil if url.blank?

      # 添付ファイルのURLパターンかチェック
      # 例: /attachments/:attachment_id
      match = url.match(%r{^/attachments/([^/]+)$})
      return nil unless match

      match[1]
    end

    sig { params(attachment: AttachmentRecord).returns(T::Boolean) }
    private def can_display_inline_image?(attachment)
      # ファイル拡張子を取得
      filename = attachment.filename
      return false unless filename

      extension = File.extname(filename).downcase.delete(".")
      INLINE_IMAGE_FORMATS.include?(extension)
    end

    sig { params(attachment: AttachmentRecord).returns(T::Boolean) }
    private def can_display_inline_video?(attachment)
      # ファイル拡張子を取得
      filename = attachment.filename
      return false unless filename

      extension = File.extname(filename).downcase.delete(".")
      INLINE_VIDEO_FORMATS.include?(extension)
    end

    sig { params(element: Selma::HTML::Element, attachment: AttachmentRecord).void }
    private def convert_to_download_link(element, attachment)
      # img要素をa要素に変換
      filename = attachment.filename || "ファイル"

      # リンクとして表示（署名付きURLは後でJavaScriptで置換）
      link_html = <<~HTML
        <a href="#"
           data-attachment-id="#{attachment.id}"
           data-attachment-link="true"
           target="_blank"
           rel="noopener noreferrer"
           class="inline-flex items-center gap-1 text-blue-600 hover:text-blue-800 underline wikino-attachment-link">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          #{CGI.escapeHTML(filename)}
        </a>
      HTML

      # element.replaceの代わりに、after挿入してから元のelementを削除
      element.after(link_html, as: :html)
      element.remove
    end

    sig { returns(Space) }
    attr_reader :current_space
    private :current_space

    sig { returns(T.nilable(SpaceMember)) }
    attr_reader :current_space_member
    private :current_space_member
  end
end
