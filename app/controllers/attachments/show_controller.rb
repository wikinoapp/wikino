# typed: strict
# frozen_string_literal: true

module Attachments
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable

    before_action :restore_user_session

    # GET /attachments/:id
    # NOTE: このエンドポイントは添付ファイルを作ったときページ本文に挿入される
    #       このエンドポイントを `/s/:space_identifier/` 配下に置いていないのは、
    #       スペースの識別子を変えたときにページ本文内の添付ファイルのURLが無効になるのを防ぐため
    sig { void }
    def call
      attachment_record = AttachmentRecord.find(params[:attachment_id])

      # page_attachment_references を経由して添付されているページを取得
      page_attachment_refs = PageAttachmentReferenceRecord
        .preload(page_record: :topic_record)
        .where(attachment_record:)

      # 添付ファイルを参照している全てのページを取得
      pages_with_topics = page_attachment_refs.map do |ref|
        page_record = ref.page_record
        next nil if page_record.nil?

        topic_record = page_record.topic_record
        next nil if topic_record.nil?

        {page: page_record, topic: topic_record}
      end.compact

      # アクセス権限をチェック
      if pages_with_topics.any?
        # 全てのページが公開トピックかチェック
        all_pages_public = pages_with_topics.all? do |item|
          item[:topic].visibility_public?
        end

        if all_pages_public
          # 全て公開トピックの場合は誰でもアクセス可能
        else
          # 公開でないトピックがある場合は、スペースメンバーのみアクセス可能
          space_record = attachment_record.space_record
          if space_record && current_user_record
            is_space_member = SpaceMemberRecord.exists?(
              space_id: space_record.id,
              user_id: current_user_record.id,
              active: true
            )
            unless is_space_member
              render_404
              return
            end
          elsif !current_user_record
            # ログインしていない場合はアクセス拒否
            render_404
            return
          end
        end
      else
        # ページに関連付けられていない添付ファイルの場合
        # スペースメンバーのみアクセス可能
        space_record = attachment_record.space_record
        if space_record && current_user_record
          is_space_member = SpaceMemberRecord.exists?(
            space_id: space_record.id,
            user_id: current_user_record.id,
            active: true
          )
          unless is_space_member
            render_404
            return
          end
        else
          render_404
          return
        end
      end

      # Active Storageの署名付きURLを生成してリダイレクト
      active_storage_attachment = attachment_record.active_storage_attachment_record
      blob = active_storage_attachment&.blob
      if blob
        # Active Storageのサービス署名付きURLを使用
        redirect_to(blob.url, allow_other_host: true)
      else
        render_404
      end
    end
  end
end
