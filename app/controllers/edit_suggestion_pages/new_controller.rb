# typed: strict
# frozen_string_literal: true

module EditSuggestionPages
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicAware

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      topic_record = space_record.topic_records.find_by!(number: params[:topic_number])
      topic_policy = topic_policy_for(topic_record:)

      # 編集提案の作成権限をチェック
      unless topic_policy.can_create_edit_suggestion?
        return render_404
      end

      space_member_record = current_user_record!.space_member_record(space_record:)

      # 現在のユーザーが作成した下書き/オープンの編集提案を取得
      existing_edit_suggestions = EditSuggestionRecord
        .open_or_draft
        .where(topic_id: topic_record.id, created_space_member_id: space_member_record.not_nil!.id)
        .order(created_at: :desc)

      page_record = params[:page_number].present? ? space_record.page_records.find_by(number: params[:page_number]) : nil

      # 編集中のページデータを受け取る
      form = EditSuggestionPages::CreateForm.new(
        page_title: params[:page_title],
        page_body: params[:page_body]
      )

      space = SpaceRepository.new.to_model(space_record:)
      topic = TopicRepository.new.to_model(topic_record:)
      page = page_record ? PageRepository.new.to_model(page_record:) : nil
      existing_edit_suggestions_models = existing_edit_suggestions.map { |record| EditSuggestionRepository.new.to_model(edit_suggestion_record: record) }

      render EditSuggestionPages::NewView.new(
        form:,
        space:,
        topic:,
        page:,
        existing_edit_suggestions: existing_edit_suggestions_models
      ), layout: false
    end
  end
end
