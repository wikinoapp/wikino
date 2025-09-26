# typed: strict
# frozen_string_literal: true

module EditSuggestionPages
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicAware
    include ControllerConcerns::EditSuggestionRenderable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      page_record = space_record.page_records.find_by!(number: params[:page_number])
      topic_record = page_record.topic_record.not_nil!
      topic_policy = topic_policy_for(topic_record:)

      unless topic_policy.can_create_edit_suggestion?
        return render_404
      end

      space_member_record = current_user_record!.space_member_record(space_record:)

      form = EditSuggestionPages::CreateForm.new(
        form_params.merge(space_member_record:, topic_record:)
      )

      if form.invalid?
        view = build_edit_suggestion_page_view(
          form:,
          page_record:,
          space_member_record:
        )
        return render_component(view, status: :unprocessable_entity)
      end

      # 既存の編集提案にページを追加
      # フォームのバリデーションで取得したedit_suggestion_recordを使用
      EditSuggestionPages::AddService.new.call(
        edit_suggestion_record: form.edit_suggestion_record.not_nil!,
        space_member_record: space_member_record.not_nil!,
        page_record:,
        page_title: form.page_title.not_nil!,
        page_body: form.page_body.not_nil!
      )

      flash[:notice] = t("messages.edit_suggestion_pages.added")
      # TODO: ShowControllerが実装されたらedit_suggestion_pathに変更する
      redirect_to topic_edit_suggestion_list_path(
        space_identifier: space_record.identifier,
        topic_number: topic_record.number
      )
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:edit_suggestion_pages_create_form), ActionController::Parameters).permit(
        :edit_suggestion_id,
        :page_title,
        :page_body
      )
    end
  end
end
