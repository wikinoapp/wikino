# typed: strict
# frozen_string_literal: true

module Pages
  class UpdateService < ApplicationService
    class Result < T::Struct
      const :page_record, PageRecord
    end

    sig do
      params(
        space_member_record: SpaceMemberRecord,
        page_record: PageRecord,
        topic_record: TopicRecord,
        title: String,
        body: String
      ).returns(Result)
    end
    def call(space_member_record:, page_record:, topic_record:, title:, body:)
      now = Time.current

      page_record.attributes = {
        topic_record:,
        title:,
        body:,
        modified_at: now
      }
      page_record.published_at = now if page_record.published_at.nil?

      updated_page_record = ActiveRecord::Base.transaction do
        page_record.save!
        page_record.add_editor!(editor_record: space_member_record)
        page_record.create_revision!(editor_record: space_member_record, body:)
        page_record.link!(editor_record: space_member_record)
        space_member_record.destroy_draft_page!(page_record:)

        # ページ本文から添付ファイルIDを検知し、参照を更新
        update_attachment_references!(page_record:, body:)

        page_record
      end

      Result.new(page_record: updated_page_record)
    end

    private

    # ページ本文から添付ファイルIDを抽出し、page_attachment_referencesレコードを更新
    sig { params(page_record: PageRecord, body: String).void }
    def update_attachment_references!(page_record:, body:)
      # 本文から添付ファイルのIDを抽出
      # URLパターン: /attachments/:attachment_id
      # imgタグのsrc属性やaタグのhref属性から抽出
      attachment_ids = extract_attachment_ids(body)

      # 現在の参照を取得
      current_references = page_record.page_attachment_reference_records
      current_attachment_ids = current_references.pluck(:attachment_id)

      # 新しく追加される添付ファイル
      new_attachment_ids = attachment_ids - current_attachment_ids

      # 削除される添付ファイル
      removed_attachment_ids = current_attachment_ids - attachment_ids

      # 新しい参照を作成
      new_attachment_ids.each do |attachment_id|
        # 添付ファイルが実際に存在するか確認
        if AttachmentRecord.exists?(id: attachment_id, space_id: page_record.space_id)
          PageAttachmentReferenceRecord.create!(
            page_id: page_record.id,
            attachment_id:
          )
        end
      end

      # 不要な参照を削除
      if removed_attachment_ids.any?
        PageAttachmentReferenceRecord.where(
          page_id: page_record.id,
          attachment_id: removed_attachment_ids
        ).destroy_all
      end

      nil
    end

    # 本文から添付ファイルIDを抽出
    sig { params(body: String).returns(T::Array[String]) }
    def extract_attachment_ids(body)
      attachment_ids = T.let([], T::Array[String])

      # imgタグのsrc属性から抽出
      # <img src="/attachments/attachment_id">
      img_pattern = %r{<img[^>]+src=["'](/attachments/([^/"']+))["'][^>]*>}
      body.scan(img_pattern) do |_full_url, attachment_id|
        attachment_ids << attachment_id if attachment_id
      end

      # aタグのhref属性から抽出
      # <a href="/attachments/attachment_id">
      link_pattern = %r{<a[^>]+href=["'](/attachments/([^/"']+))["'][^>]*>}
      body.scan(link_pattern) do |_full_url, attachment_id|
        attachment_ids << attachment_id if attachment_id
      end

      # 重複を削除
      attachment_ids.uniq
    end
  end
end
