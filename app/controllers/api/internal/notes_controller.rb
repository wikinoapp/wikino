# frozen_string_literal: true

module Api
  module Internal
    class NotesController < ApplicationController
      before_action :authenticate_user!

      def index
        note_entities = InternalApi::NoteList::NotesRepository.new(graphql_client: graphql_client).call(q: params[:q])

        render json: {
          notes: note_entities.map do |note_entity|
            {
              title: note_entity.title
            }
          end
        }
      end

      def update
        fetched_note_entity = UpdateNote::NoteRepository.
          new(graphql_client: graphql_client).
          call(database_id: params[:note_id])

        @note_entity, @error_code = UpdateNoteRepository.
          new(graphql_client: graphql_client).
          call(id: fetched_note_entity.id, body: note_params[:body])
      end

      private

      def note_params
        params.require(:note).permit(:body)
      end
    end
  end
end
