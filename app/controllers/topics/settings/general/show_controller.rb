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
          space = SpaceRecord.find_by_identifier!(params[:space_identifier])
          space_viewer = Current.viewer!.space_viewer!(space:)
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
            current_user_entity: Current.viewer!.user_entity,
            topic_entity:,
            form:
          )
        end
      end
    end
  end
end
