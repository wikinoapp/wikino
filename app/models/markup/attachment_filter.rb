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

    # @override
    sig { params(context: T::Hash[Symbol, T.untyped], result: T::Hash[Symbol, T.untyped]).void }
    def initialize(context: {}, result: {})
      super
      @current_space = T.let(context[:current_space], SpaceRecord)
      @current_space_member = T.let(context[:current_space_member], T.nilable(SpaceMemberRecord))
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
      attachment = AttachmentRecord.find_by(id: attachment_id, space_record: current_space)
      return unless attachment

      # インライン表示可能な画像形式かチェック
      if can_display_inline?(attachment)
        # 署名付きURLを生成
        signed_url = generate_signed_url(attachment_id)

        # スペースメンバーの場合のみa要素で囲む
        if signed_url
          # 署名付きURLが生成できた場合は、a要素で囲む
          # 新しいHTMLを作成
          link_html = <<~HTML
            <a href="#{signed_url}" target="_blank" rel="noopener noreferrer" class="inline-block">
              <img src="#{signed_url}" class="max-w-full" />
            </a>
          HTML

          # element.replaceの代わりに、after挿入してから元のelementを削除
          element.after(link_html, as: :html)
          element.remove
        else
          # 署名付きURLが生成できない場合（非メンバー）は、img要素のみ
          element["class"] = "max-w-full"
        end
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

      # 署名付きURLを生成
      signed_url = generate_signed_url(attachment_id)
      element["href"] = signed_url if signed_url

      # 新規タブで開く
      element["target"] = "_blank"
      element["rel"] = "noopener noreferrer"
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

    sig { params(attachment_id: String).returns(T.nilable(String)) }
    private def generate_signed_url(attachment_id)
      # 権限チェック
      return nil unless can_access_attachment?

      # AttachmentRecordを取得
      attachment = AttachmentRecord.find_by(id: attachment_id, space_record: current_space)
      return nil unless attachment

      # 署名付きURLを生成（1時間の有効期限）
      attachment.generate_signed_url(
        space_member_record: current_space_member,
        expires_in: 1.hour
      )
    rescue => e
      Rails.logger.error("Failed to generate signed URL for attachment #{attachment_id}: #{e.message}")
      nil
    end

    sig { returns(T::Boolean) }
    private def can_access_attachment?
      # スペースメンバーであれば添付ファイルにアクセス可能
      current_space_member.present?
    end

    sig { params(attachment: AttachmentRecord).returns(T::Boolean) }
    private def can_display_inline?(attachment)
      # ファイル拡張子を取得
      filename = attachment.filename
      return false unless filename

      extension = File.extname(filename).downcase.delete(".")
      INLINE_IMAGE_FORMATS.include?(extension)
    end

    sig { params(element: Selma::HTML::Element, attachment: AttachmentRecord).void }
    private def convert_to_download_link(element, attachment)
      # img要素をa要素に変換
      filename = attachment.filename || "ファイル"

      # 署名付きURLを生成
      signed_url = generate_signed_url(attachment.id)

      # リンクとして表示
      link_html = <<~HTML
        <a href="#{signed_url || element["src"]}"
           target="_blank"
           rel="noopener noreferrer"
           class="inline-flex items-center gap-1 text-blue-600 hover:text-blue-800 underline">
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

    sig { returns(SpaceRecord) }
    attr_reader :current_space

    sig { returns(T.nilable(SpaceMemberRecord)) }
    attr_reader :current_space_member
  end
end
