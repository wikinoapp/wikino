# typed: strict
# frozen_string_literal: true

module EditSuggestionPages
  class CreateController < ApplicationController
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
      edit_suggestion_record = EditSuggestionRecord
        .open_or_draft
        .where(topic_id: topic_record.id, created_space_member_id: space_member_record.not_nil!.id)
        .find_by(id: params[:id])

      # 編集提案が存在しない、または権限がない場合
      unless edit_suggestion_record
        return render_404
      end

      page_record = params[:page_number].present? ? space_record.page_records.find_by(number: params[:page_number]) : nil

      form = EditSuggestionPages::CreateForm.new(
        form_params.merge(
          space_member_record:,
          page_record:,
          edit_suggestion_record:
        )
      )

      if form.invalid?
        return render(
          turbo_stream: helpers.turbo_stream.update(
            "edit-suggestion-form-errors",
            partial: "shared/form_errors",
            locals: {form:}
          ),
          status: :unprocessable_entity
        )
      end

      # 既存の編集提案にページを追加
      EditSuggestionPages::AddService.new.call(
        edit_suggestion_record:,
        space_member_record: space_member_record.not_nil!,
        page_record:,
        page_title: form.page_title.not_nil!,
        page_body: form.page_body.not_nil!
      )

      # Turbo Streamリクエストの場合はリダイレクト用のレスポンスを返す
      if request.headers["Accept"]&.include?("text/vnd.turbo-stream.html")
        flash.now[:notice] = t("messages.edit_suggestion_pages.added")
        render turbo_stream: [
          helpers.turbo_stream.redirect(edit_suggestion_path(
            space_identifier: space_record.identifier,
            topic_number: topic_record.number,
            id: edit_suggestion_record.id
          ))
        ]
      else
        flash[:notice] = t("messages.edit_suggestion_pages.added")
        redirect_to edit_suggestion_path(
          space_identifier: space_record.identifier,
          topic_number: topic_record.number,
          id: edit_suggestion_record.id
        )
      end
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:edit_suggestion_pages_create_form), ActionController::Parameters).permit(
        :page_title,
        :page_body
      )
    end
  end
end
