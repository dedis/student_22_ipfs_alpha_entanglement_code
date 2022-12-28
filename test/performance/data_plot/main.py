import only_data
import data_and_parity
import different_nb_nodes
import different_file_size


def main():
    save_to_file = True

    # ONLY DATA MISSING
    only_data.download_overhead(save_to_file)
    only_data.memory_overhead(save_to_file)

    # DATA ALL MISSING, PART OF PARITY MISSING
    data_and_parity.download_overhead(save_to_file)
    data_and_parity.memory_overhead(save_to_file)
    data_and_parity.recovery_likelihood(save_to_file)
    data_and_parity.best_effort(save_to_file)

    # DIFFERENT FILE SIZE
    different_file_size.download_overhead(save_to_file)
    different_file_size.memory_overhead(save_to_file)
    different_file_size.recovery_likelihood(save_to_file)
    different_file_size.best_effort(save_to_file)

    # DIFFERENT NUMBER OF NODES
    different_nb_nodes.download_overhead(save_to_file)
    different_nb_nodes.memory_overhead(save_to_file)
    different_nb_nodes.recovery_likelihood(save_to_file)
    different_nb_nodes.best_effort(save_to_file)


if __name__ == "__main__":
    main()
