# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module NotebookSettable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { void }
    private def set_notebook
      @notebook = T.let(viewer!.space.notebooks.kept.find_by!(number: params[:notebook_number]), T.nilable(Notebook))
      authorize(@notebook, :show?)
    end
  end
end
