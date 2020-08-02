# frozen_string_literal: true

module Api
  module Internal
    class NotesController < ApplicationController
      before_action :authenticate_user!

      def index
        @note_entities = InternalApi::NoteList::FetchNotesRepository.new(graphql_client: graphql_client).call(q: params[:q])
      end

      def update
        note_entity = UpdateNote::FetchNoteRepository.
          new(graphql_client: graphql_client).
          call(database_id: params[:note_id])

        updated_note_entity, mutation_error_entities = UpdateNoteRepository.
          new(graphql_client: graphql_client).
          call(id: note_entity.id, body: note_params[:body])

        render json: { updated_at: updated_note_entity.updated_at }
      end

      private

      def note_params
        params.require(:note).permit(:body)
      end
    end
  end
end
