# frozen_string_literal: true

class NotesController < ApplicationController
  before_action :authenticate_user!

  def index
    @note_entities, @page_info_entity = NoteList::FetchNotesRepository.
      new(graphql_client: graphql_client).
      call(
        pagination: Pagination.new(before: params[:before], after: params[:after], per: 30)
      )
  end

  def show
    note_entity = NoteDetail::FetchNoteRepository.
      new(graphql_client: graphql_client).
      call(database_id: params[:note_id])

    redirect_to edit_note_path(note_entity.database_id)
  end

  def new
    note_entity, mutation_error_entities = CreateNoteRepository.new(graphql_client: graphql_client).call

    if mutation_error_entities
      return redirect_to root_path, alert: t("messages._common.unexpected_error")
    end

    redirect_to edit_note_path(note_entity.database_id)
  end

  def edit
    @note_entity = NoteDetail::FetchNoteRepository.
      new(graphql_client: graphql_client).
      call(database_id: params[:note_id])
  end

  private

  def note_params
    params.require(:note).permit(:body)
  end
end
