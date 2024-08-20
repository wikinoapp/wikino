# typed: strict
# frozen_string_literal: true

module Cards
  class NotebookCardComponent < ApplicationComponent
    sig { params(notebook: Notebook, class_name: String).void }
    def initialize(notebook:, class_name: "")
      @notebook = notebook
      @class_name = class_name
    end

    sig { returns(Notebook) }
    attr_reader :notebook
    private :notebook

    sig { returns(String) }
    attr_reader :class_name
    private :class_name
  end
end
