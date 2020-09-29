
def argument_reader(arguments = nil, &block)
    if $stdin.tty?
        argv_argument_reader(arguments, &block)
    else
        stdin_arguments_reader(&block)
    end
end

def argv_argument_reader(arguments = nil, &block)
    (arguments || ARGV).each do |argument|
        block.call argument
    end
end

def stdin_arguments_reader(&block)
    $stdin.each_line do |line|
        line.split.each do |element|
            block.call element
        end
    end
end
