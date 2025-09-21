# typed: true
# frozen_string_literal: true

module EditSuggestions
  class IndexController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicAware

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      current_user_record&.space_member_record(space_record:)
      topic_record = space_record.topic_record_by_number!(params[:topic_number])
      topic_policy = topic_policy_for(topic_record:)

      unless topic_policy.can_show_topic?(topic_record:)
        return render_404
      end

      # フィルター状態の判定（オープン/クローズ）
      # オープン表示：下書き・オープンステータスの編集提案
      # クローズ表示：反映済み・クローズステータスの編集提案
      filter_state = params[:state].presence || "open"
      is_open_filter = filter_state == "open"

      # 編集提案の取得とフィルタリング
      edit_suggestion_records_relation = EditSuggestionRecord
        .where(topic_record:)
        .preload(:created_space_member_record, :edit_suggestion_page_records)

      edit_suggestion_records_relation = if is_open_filter
        edit_suggestion_records_relation
          .where(status: [EditSuggestionStatus::Draft.serialize, EditSuggestionStatus::Open.serialize])
      else
        edit_suggestion_records_relation
          .where(status: [EditSuggestionStatus::Applied.serialize, EditSuggestionStatus::Closed.serialize])
      end

      edit_suggestion_records = edit_suggestion_records_relation
        .order(created_at: :desc)
        .limit(100)

      edit_suggestions = EditSuggestionRepository.new.to_models(
        edit_suggestion_records:
      )

      topic = TopicRepository.new.to_model(
        topic_record:,
        can_update: topic_policy.can_update_topic?(topic_record:),
        can_create_page: topic_policy.can_create_page?(topic_record:)
      )

      render_component EditSuggestions::IndexView.new(
        current_user:,
        topic:,
        edit_suggestions:,
        filter_state:
      )
    end
  end
end
