# typed: true
# frozen_string_literal: true

module Topics
  module Settings
    module General
      class ShowController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
          space_member_record = current_user!.space_member_record(space_record:)
          topic = space_viewer.showable_topics.find_by!(number: params[:topic_number])
          topic_entity = topic.to_entity(space_viewer:)

          unless topic_entity.viewer_can_update?
            return render_404
          end

          form = EditTopicForm.new(
            name: topic_entity.name,
            description: topic_entity.description,
            visibility: topic_entity.visibility.serialize
          )

          render Topics::Settings::General::ShowView.new(
            current_user: current_user!,
            topic_entity:,
            form:
          )
        end
      end
    end
  end
end
