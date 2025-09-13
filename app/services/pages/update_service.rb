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

      # body_htmlを生成（プレースホルダー形式）
      topic = TopicRepository.new.to_model(topic_record:)
      space = SpaceRepository.new.to_model(space_record: page_record.space_record!)
      space_member = SpaceMemberRepository.new.to_model(space_member_record:)

      body_html = Markup.new(
        current_topic: topic,
        current_space: space,
        current_space_member: space_member
      ).render_html(text: body)

      page_record.attributes = {
        topic_record:,
        title:,
        body:,
        body_html:,
        modified_at: now
      }
      page_record.published_at = now if page_record.published_at.nil?

      updated_page_record = ActiveRecord::Base.transaction do
        page_record.save!
        page_record.add_editor!(editor_record: space_member_record)
        page_record.create_revision!(editor_record: space_member_record, body:, body_html:)
        page_record.link!(editor_record: space_member_record)
        space_member_record.destroy_draft_page!(page_record:)

        # topic_membersレコードのlast_page_modified_atを更新
        topic_member_record = TopicMemberRecord.find_by!(
          topic_id: topic_record.id,
          space_member_id: space_member_record.id
        )
        topic_member_record.update_last_page_modified_at!(time: now)

        # ページ本文から添付ファイルIDを検知し、参照を更新
        page_record.update_attachment_references!(body:)

        # 1行目の画像IDを抽出（featured画像として）
        featured_image_id = page_record.extract_featured_image_id

        if featured_image_id
          # 同じスペースの添付ファイルか確認
          attachment = AttachmentRecord.find_by(
            id: featured_image_id,
            space_id: page_record.space_id
          )

          if attachment
            # featured_image_attachment_idを更新
            page_record.update!(featured_image_attachment_id: attachment.id)
          else
            # 画像が見つからない場合はnullに設定
            page_record.update!(featured_image_attachment_id: nil)
          end
        else
          # 1行目に画像がない場合はnullに設定
          page_record.update!(featured_image_attachment_id: nil)
        end

        page_record
      end

      Result.new(page_record: updated_page_record)
    end
  end
end
