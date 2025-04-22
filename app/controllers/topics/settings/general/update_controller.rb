# typed: true
# frozen_string_literal: true

module Topics
  module Settings
    module General
      class UpdateController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          current_space_member = current_user!.current_space_member(space_record:)
          topic = space_viewer.showable_topics.find_by!(number: params[:topic_number])
          topic_entity = topic.to_entity(space_viewer:)

          unless topic_entity.viewer_can_update?
            return render_404
          end

          form = EditTopicForm.new(form_params.merge(topic_record: topic))

          if form.invalid?
            return render(
              Topics::Settings::General::ShowView.new(
                current_user: current_user!,
                topic_entity:,
                form:
              ),
              status: :unprocessable_entity
            )
          end

          UpdateTopicService.new.call(form:)

          flash[:notice] = t("messages.topics.updated")
          redirect_to topic_settings_general_path(space.identifier, topic.number)
        end

        sig { returns(ActionController::Parameters) }
        private def form_params
          T.cast(params.require(:edit_topic_form), ActionController::Parameters).permit(
            :name,
            :description,
            :visibility
          )
        end
      end
    end
  end
end
