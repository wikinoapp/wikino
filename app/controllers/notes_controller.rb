# frozen_string_literal: true

class NotesController < ApplicationController
  before_action :authenticate_user!

  def index
    @note_entities, @page_info_entity = NoteList::FetchNotesRepository.
      new(graphql_client: graphql_client).
      fetch(
        pagination: Pagination.new(before: params[:before], after: params[:after], per: 30)
      )
  end

  def show
    @note_entity = NoteDetail::FetchNoteRepository.
      new(graphql_client: graphql_client).
      fetch(database_id: params[:note_id])
  end

  def new
    @note = current_user.notes.new
  end

  def create
    note_entity, mutation_error_entities = CreateNoteRepository.new(graphql_client: graphql_client).call(params: note_params)

    if mutation_error_entities
      @note = current_user.notes.new(note_params)
      mutation_error_entities.each do |error_entity|
        @note.errors.add(:mutation_error, error_entity.message)
      end

      return render(:new)
    end

    redirect_to note_detail_path(note_entity.database_id)
  end

  private

  def note_params
    params.require(:note).permit(:body)
  end
end
