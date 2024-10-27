# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module PageSettable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { void }
    private def set_page
      @page = T.let(Current.space!.pages.find_by!(number: params[:page_number]), T.nilable(Page))
    end
  end
end
