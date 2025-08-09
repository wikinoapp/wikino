# typed: strict
# frozen_string_literal: true

module MarkupFilters
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

    private

    sig { params(element: Selma::HTML::Element).void }
    def handle_image(element)
      src = element["src"]
      return if src.nil? || src.empty?

      # 添付ファイルのURLパターンかチェック
      # 例: /s/:space_identifier/attachments/:attachment_id
      match = src.match(%r{^/s/[^/]+/attachments/([^/]+)$})
      return unless match

      attachment_id = match[1]
      
      # AttachmentRecordを取得してファイル形式を確認
      attachment = AttachmentRecord.find_by(id: attachment_id, space: current_space)
      return unless attachment

      # インライン表示可能な画像形式かチェック
      if can_display_inline?(attachment)
        # 署名付きURLを生成
        signed_url = generate_signed_url(attachment_id)
        element["src"] = signed_url if signed_url

        # 画像のクリックで新規タブ表示するための処理
        element["class"] = "cursor-pointer hover:opacity-90 transition-opacity max-w-full"
        element["data-controller"] = "attachment-viewer"
        element["data-action"] = "click->attachment-viewer#open"
        element["data-attachment-viewer-url-value"] = signed_url || src
      else
        # インライン表示不可の場合はリンクに変換
        convert_to_download_link(element, attachment)
      end
    end

    sig { params(element: Selma::HTML::Element).void }
    def handle_link(element)
      href = element["href"]
      return if href.nil? || href.empty?

      # 添付ファイルのURLパターンかチェック
      match = href.match(%r{^/s/[^/]+/attachments/([^/]+)$})
      return unless match

      attachment_id = match[1]
      
      # 署名付きURLを生成
      signed_url = generate_signed_url(attachment_id)
      element["href"] = signed_url if signed_url

      # 新規タブで開く
      element["target"] = "_blank"
      element["rel"] = "noopener noreferrer"
    end

    sig { params(attachment_id: String).returns(T.nilable(String)) }
    def generate_signed_url(attachment_id)
      # 権限チェック
      return nil unless can_access_attachment?

      # AttachmentRecordを取得
      attachment = AttachmentRecord.find_by(id: attachment_id, space: current_space)
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
    def can_access_attachment?
      # スペースメンバーであれば添付ファイルにアクセス可能
      current_space_member.present?
    end

    sig { params(attachment: AttachmentRecord).returns(T::Boolean) }
    def can_display_inline?(attachment)
      return false unless attachment.active_storage_attachment&.blob

      # ファイル拡張子を取得
      filename = attachment.active_storage_attachment.blob.filename.to_s
      extension = File.extname(filename).downcase.delete(".")
      
      INLINE_IMAGE_FORMATS.include?(extension)
    end

    sig { params(element: Selma::HTML::Element, attachment: AttachmentRecord).void }
    def convert_to_download_link(element, attachment)
      # img要素をa要素に変換
      filename = attachment.active_storage_attachment&.blob&.filename&.to_s || "ファイル"
      
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
      
      element.replace(link_html, as: :html)
    end

    sig { returns(SpaceRecord) }
    attr_reader :current_space

    sig { returns(T.nilable(SpaceMemberRecord)) }
    attr_reader :current_space_member
  end
end